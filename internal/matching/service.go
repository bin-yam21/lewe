package matching

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/items"
	"github.com/yeabt/lewe/internal/validator"
)

// allowedExchangeMethods are the concrete methods a match can settle on.
// Items may advertise "either"; a match must pick one.
var allowedExchangeMethods = []string{"in_person", "shipping"}

// Service contains matching business logic.
type Service struct {
	repo     *Repository
	itemRepo *items.Repository
}

// NewService creates a new matching service.
func NewService(repo *Repository, itemRepo *items.Repository) *Service {
	return &Service{repo: repo, itemRepo: itemRepo}
}

// FindMatchesForItem discovers and creates matches for a user's item.
func (s *Service) FindMatchesForItem(ctx context.Context, userID, itemID pgtype.UUID) (*FoundMatchesResponse, error) {
	// Verify the item exists and belongs to the user
	item, _, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item.UserID != userID {
		return nil, items.ErrNotItemOwner
	}
	if item.Status != "active" {
		return nil, errors.New("item is not active")
	}

	// Find potential matches (items not yet matched)
	potentialIDs, err := s.repo.FindPotentialMatches(ctx, itemID, userID)
	if err != nil {
		return nil, err
	}

	// Create match records for each potential match
	var matchResponses []MatchResponse
	for _, otherID := range potentialIDs {
		match, err := s.repo.CreateMatch(ctx, itemID, otherID)
		if err != nil {
			continue // skip on error (e.g. race condition)
		}
		// Fetch the full match with item details
		mwi, err := s.repo.GetByID(ctx, match.ID)
		if err != nil {
			continue
		}
		matchResponses = append(matchResponses, toMatchResponse(mwi))
	}

	// Also include existing non-cancelled matches for this item
	existing, err := s.repo.FindMatches(ctx, itemID, userID)
	if err != nil {
		return nil, err
	}

	// Build a set of already-included match IDs
	seen := make(map[string]bool)
	for _, mr := range matchResponses {
		seen[mr.ID] = true
	}
	for _, mwi := range existing {
		id := uuidToString(mwi.Match.ID)
		if !seen[id] {
			matchResponses = append(matchResponses, toMatchResponse(&mwi))
		}
	}

	if matchResponses == nil {
		matchResponses = []MatchResponse{}
	}

	return &FoundMatchesResponse{
		Matches: matchResponses,
		Count:   len(matchResponses),
	}, nil
}

// OfferTrade proposes a swap directly to an item's owner.
//
// This is the path for someone browsing who sees something they want: rather
// than publishing a listing and waiting for the mutual-wants query to pair
// them, they hand over one of their own items right now. The offered item may
// be private — never published to the feed at all.
//
// Everything downstream is unchanged: the offer is a pending match, so the
// owner accepts or declines it, both sides confirm, and both sides rate.
func (s *Service) OfferTrade(ctx context.Context, userID, targetItemID, offerItemID pgtype.UUID, message *string) (*MatchResponse, error) {
	if targetItemID == offerItemID {
		return nil, ErrInvalidOffer
	}

	target, _, err := s.itemRepo.GetByID(ctx, targetItemID)
	if err != nil {
		return nil, err
	}
	offered, _, err := s.itemRepo.GetByID(ctx, offerItemID)
	if err != nil {
		return nil, err
	}

	// You offer your own item, for someone else's.
	if offered.UserID != userID {
		return nil, items.ErrNotItemOwner
	}
	if target.UserID == userID {
		return nil, ErrOwnItem
	}

	// The target has to still be available; the offered item has to be one the
	// user can actually part with. "private" is allowed — that is the point.
	if target.Status != "active" {
		return nil, ErrItemUnavailable
	}
	if offered.Status != "active" && offered.Status != "private" {
		return nil, ErrItemUnavailable
	}

	// An existing live match between these two items already covers this.
	if existing, err := s.repo.FindBetween(ctx, offerItemID, targetItemID); err == nil {
		mwi, err := s.repo.GetByID(ctx, existing.ID)
		if err != nil {
			return nil, err
		}
		resp := toMatchResponse(mwi)
		return &resp, ErrOfferExists
	} else if !errors.Is(err, ErrMatchNotFound) {
		return nil, err
	}

	match, err := s.repo.CreateDirectOffer(ctx, offerItemID, targetItemID, message)
	if err != nil {
		return nil, err
	}

	return s.freshResponse(ctx, match.ID)
}

// RespondToMatch accepts or declines a match. When accepting, the responder may
// also propose (or change) the exchange method for the trade.
func (s *Service) RespondToMatch(ctx context.Context, userID, matchID pgtype.UUID, req RespondRequest) (*MatchResponse, error) {
	if req.Accept && req.ExchangeMethod != nil {
		errs := make(validator.Errors)
		validator.ValidateOneOf(errs, "exchange_method", *req.ExchangeMethod, allowedExchangeMethods)
		if errs.HasErrors() {
			return nil, errs
		}
	}

	mwi, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// Determine which side the user is on
	side := s.userSide(mwi, userID)
	if side == "" {
		return nil, ErrNotMatchParty
	}

	if !req.Accept {
		// Decline → cancel
		if err := s.repo.UpdateStatus(ctx, matchID, "cancelled"); err != nil {
			return nil, err
		}
		return s.freshResponse(ctx, matchID)
	}

	// Accept — validate current status allows acceptance
	switch mwi.Match.Status {
	case "pending":
		if side == "a" {
			if err := s.repo.UpdateStatus(ctx, matchID, "accepted_a"); err != nil {
				return nil, err
			}
			mwi.Match.Status = "accepted_a"
		} else {
			if err := s.repo.UpdateStatus(ctx, matchID, "accepted_b"); err != nil {
				return nil, err
			}
			mwi.Match.Status = "accepted_b"
		}
	case "accepted_a":
		if side == "b" {
			// Both sides accepted → confirmed
			if err := s.repo.UpdateStatus(ctx, matchID, "confirmed"); err != nil {
				return nil, err
			}
			mwi.Match.Status = "confirmed"
			// Mark both items as matched
			_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemAID, "matched")
			_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemBID, "matched")
		} else {
			return nil, ErrInvalidTransition // side A already accepted
		}
	case "accepted_b":
		if side == "a" {
			// Both sides accepted → confirmed
			if err := s.repo.UpdateStatus(ctx, matchID, "confirmed"); err != nil {
				return nil, err
			}
			mwi.Match.Status = "confirmed"
			_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemAID, "matched")
			_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemBID, "matched")
		} else {
			return nil, ErrInvalidTransition // side B already accepted
		}
	default:
		return nil, ErrInvalidTransition
	}

	// Record the agreed exchange method, if the responder supplied one.
	// The later responder's choice wins — the match is not final until both
	// sides have accepted anyway.
	if req.ExchangeMethod != nil {
		if err := s.repo.SetExchangeMethod(ctx, matchID, *req.ExchangeMethod); err != nil {
			return nil, err
		}
	}

	return s.freshResponse(ctx, matchID)
}

// CompleteMatch marks a user's side of the exchange as completed.
// When both sides confirm, the match status moves to "completed".
func (s *Service) CompleteMatch(ctx context.Context, userID, matchID pgtype.UUID) (*MatchResponse, error) {
	mwi, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	side := s.userSide(mwi, userID)
	if side == "" {
		return nil, ErrNotMatchParty
	}

	if mwi.Match.Status != "confirmed" {
		return nil, ErrInvalidTransition
	}

	// Set this side's confirmation
	if err := s.repo.SetConfirmation(ctx, matchID, side, true); err != nil {
		return nil, err
	}

	// Check if both sides confirmed
	confA, confB, err := s.repo.GetConfirmations(ctx, matchID)
	if err != nil {
		return nil, err
	}

	if confA && confB {
		// Both confirmed → completed
		if err := s.repo.UpdateStatus(ctx, matchID, "completed"); err != nil {
			return nil, err
		}
		mwi.Match.Status = "completed"
		// Mark both items as exchanged
		_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemAID, "exchanged")
		_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemBID, "exchanged")
	}

	return s.freshResponse(ctx, matchID)
}

// CancelMatch cancels a match. Only participants can cancel.
func (s *Service) CancelMatch(ctx context.Context, userID, matchID pgtype.UUID) (*MatchResponse, error) {
	mwi, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	side := s.userSide(mwi, userID)
	if side == "" {
		return nil, ErrNotMatchParty
	}

	if mwi.Match.Status == "completed" || mwi.Match.Status == "cancelled" {
		return nil, ErrInvalidTransition
	}

	if err := s.repo.UpdateStatus(ctx, matchID, "cancelled"); err != nil {
		return nil, err
	}

	// Restore items to active if they were matched
	if mwi.Match.Status == "confirmed" || mwi.Match.Status == "accepted_a" || mwi.Match.Status == "accepted_b" {
		_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemAID, "active")
		_ = s.itemRepo.UpdateStatus(ctx, mwi.Match.ItemBID, "active")
	}

	return s.freshResponse(ctx, matchID)
}

// ListMyMatches returns all matches for the authenticated user.
func (s *Service) ListMyMatches(ctx context.Context, userID pgtype.UUID) (*MatchListResponse, error) {
	matches, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]MatchResponse, 0, len(matches))
	for _, mwi := range matches {
		responses = append(responses, toMatchResponse(&mwi))
	}

	return &MatchListResponse{Matches: responses}, nil
}

// GetMatch returns a single match by ID. Only participants can view.
func (s *Service) GetMatch(ctx context.Context, userID, matchID pgtype.UUID) (*MatchResponse, error) {
	mwi, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	side := s.userSide(mwi, userID)
	if side == "" {
		return nil, ErrNotMatchParty
	}

	resp := toMatchResponse(mwi)
	return &resp, nil
}

// --- helpers ---

// freshResponse re-reads the match and builds the response from it. State-changing
// endpoints must use this: the in-memory snapshot taken at the start of a transition
// still carries the old match status and, more importantly, the old *item* statuses,
// which are updated as a side effect of the transition.
func (s *Service) freshResponse(ctx context.Context, matchID pgtype.UUID) (*MatchResponse, error) {
	mwi, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	resp := toMatchResponse(mwi)
	return &resp, nil
}

// userSide determines if the user is side "a", side "b", or "" (not a participant).
func (s *Service) userSide(mwi *MatchWithItems, userID pgtype.UUID) string {
	if mwi.ItemAUserID == userID {
		return "a"
	}
	if mwi.ItemBUserID == userID {
		return "b"
	}
	return ""
}

// toMatchResponse maps a MatchWithItems to the public-facing DTO.
func toMatchResponse(mwi *MatchWithItems) MatchResponse {
	resp := MatchResponse{
		ID: uuidToString(mwi.Match.ID),
		ItemA: MatchItemSummary{
			ID:       uuidToString(mwi.Match.ItemAID),
			UserID:   uuidToString(mwi.ItemAUserID),
			Title:    mwi.ItemATitle,
			Category: mwi.ItemACategory,
			Status:   mwi.ItemAStatus,
		},
		ItemB: MatchItemSummary{
			ID:       uuidToString(mwi.Match.ItemBID),
			UserID:   uuidToString(mwi.ItemBUserID),
			Title:    mwi.ItemBTitle,
			Category: mwi.ItemBCategory,
			Status:   mwi.ItemBStatus,
		},
		Status:     mwi.Match.Status,
		Origin:     mwi.Match.Origin,
		ConfirmedA: mwi.Match.ConfirmedA,
		ConfirmedB: mwi.Match.ConfirmedB,
	}
	if mwi.Match.Message.Valid {
		resp.Message = &mwi.Match.Message.String
	}
	if mwi.Match.ExchangeMethod.Valid {
		resp.ExchangeMethod = &mwi.Match.ExchangeMethod.String
	}
	if mwi.Match.CreatedAt.Valid {
		resp.CreatedAt = mwi.Match.CreatedAt.Time
	}
	if mwi.Match.UpdatedAt.Valid {
		resp.UpdatedAt = mwi.Match.UpdatedAt.Time
	}
	return resp
}
