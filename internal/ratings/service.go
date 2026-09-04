package ratings

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeabt/lewe/internal/matching"
	"github.com/yeabt/lewe/internal/validator"
)

var (
	ErrNotMatchParticipant = errors.New("not a participant in this match")
	ErrMatchNotCompleted   = errors.New("match is not completed")
)

// Service contains rating business logic.
type Service struct {
	repo     *Repository
	matchSvc *matching.Service
}

// NewService creates a new ratings service.
func NewService(repo *Repository, matchSvc *matching.Service) *Service {
	return &Service{repo: repo, matchSvc: matchSvc}
}

// RateExchange validates and creates a rating for a completed exchange.
func (s *Service) RateExchange(ctx context.Context, userID, matchID pgtype.UUID, req RateRequest) (*RatingResponse, error) {
	// Validate input
	errs := make(validator.Errors)
	validator.ValidateRange(errs, "score", req.Score, 1, 5)
	if errs.HasErrors() {
		return nil, errs
	}

	// Get the match — this also verifies the user is a participant
	matchResp, err := s.matchSvc.GetMatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, matching.ErrNotMatchParty) {
			return nil, ErrNotMatchParticipant
		}
		return nil, err
	}

	// Match must be completed
	if matchResp.Status != "completed" {
		return nil, ErrMatchNotCompleted
	}

	// Check if already rated
	exists, err := s.repo.ExistsForMatchAndRater(ctx, matchID, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyRated
	}

	// Determine the ratee (the other party)
	var rateeID pgtype.UUID
	userIDStr := uuidToString(userID)
	if matchResp.ItemA.UserID == userIDStr {
		rateeID, _ = parseUUID(matchResp.ItemB.UserID)
	} else {
		rateeID, _ = parseUUID(matchResp.ItemA.UserID)
	}

	// Create the rating
	rating, err := s.repo.Create(ctx, matchID, userID, rateeID, req.Score, req.Comment)
	if err != nil {
		return nil, err
	}

	resp := toRatingResponse(rating)
	return &resp, nil
}

// GetMatchRatings retrieves the ratings left on a match. Only participants in
// the match may read them.
func (s *Service) GetMatchRatings(ctx context.Context, userID, matchID pgtype.UUID) (*MatchRatingsResponse, error) {
	// Verifies the match exists and that the caller is a participant.
	if _, err := s.matchSvc.GetMatch(ctx, userID, matchID); err != nil {
		if errors.Is(err, matching.ErrNotMatchParty) {
			return nil, ErrNotMatchParticipant
		}
		return nil, err
	}

	rows, err := s.repo.GetByMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}

	ratings := make([]RatingResponse, 0, len(rows))
	for _, r := range rows {
		ratings = append(ratings, toRatingResponse(&r))
	}

	return &MatchRatingsResponse{Ratings: ratings}, nil
}

// GetUserRating returns the aggregate rating for a user.
func (s *Service) GetUserRating(ctx context.Context, userID pgtype.UUID) (*UserRatingResponse, error) {
	avg, count, err := s.repo.GetUserAverage(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserRatingResponse{
		UserID:  uuidToString(userID),
		Average: avg,
		Count:   count,
	}, nil
}

// --- helpers ---

func toRatingResponse(r *RatingRow) RatingResponse {
	resp := RatingResponse{
		ID:      uuidToString(r.ID),
		MatchID: uuidToString(r.MatchID),
		RaterID: uuidToString(r.RaterID),
		RateeID: uuidToString(r.RateeID),
		Score:   r.Score,
	}
	if r.Comment.Valid {
		resp.Comment = &r.Comment.String
	}
	if r.CreatedAt.Valid {
		resp.CreatedAt = r.CreatedAt.Time
	}
	return resp
}
