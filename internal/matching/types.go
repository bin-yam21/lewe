package matching

import "time"

// --- Request DTOs ---

// RespondRequest is the payload for accepting or declining a match.
// ExchangeMethod is optional: when accepting, either party may propose or
// change how the exchange happens ("in_person" or "shipping"). It is ignored
// when declining.
type RespondRequest struct {
	Accept         bool    `json:"accept"`
	ExchangeMethod *string `json:"exchange_method,omitempty"`
}

// --- Response DTOs ---

// MatchItemSummary is a condensed item view embedded in match responses.
type MatchItemSummary struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

// OfferRequest proposes one of your items in exchange for someone else's.
type OfferRequest struct {
	OfferItemID string  `json:"offer_item_id"`
	Message     *string `json:"message,omitempty"`
}

// MatchResponse is the public-facing representation of a match.
type MatchResponse struct {
	ID    string           `json:"id"`
	ItemA MatchItemSummary `json:"item_a"`
	ItemB MatchItemSummary `json:"item_b"`
	/** "discovered" by mutual wants, or "direct" if someone offered. */
	Origin         string    `json:"origin"`
	Message        *string   `json:"message,omitempty"`
	Status         string    `json:"status"`
	ExchangeMethod *string   `json:"exchange_method,omitempty"`
	ConfirmedA     bool      `json:"confirmed_a"`
	ConfirmedB     bool      `json:"confirmed_b"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MatchListResponse wraps a list of matches.
type MatchListResponse struct {
	Matches []MatchResponse `json:"matches"`
}

// FoundMatchesResponse wraps the result of a match search.
type FoundMatchesResponse struct {
	Matches []MatchResponse `json:"matches"`
	Count   int             `json:"count"`
}
