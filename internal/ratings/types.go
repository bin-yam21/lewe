package ratings

import "time"

// --- Request DTOs ---

// RateRequest is the payload for rating an exchange partner.
type RateRequest struct {
	Score   int     `json:"score"`
	Comment *string `json:"comment,omitempty"`
}

// --- Response DTOs ---

// RatingResponse is the public-facing representation of a rating.
type RatingResponse struct {
	ID        string    `json:"id"`
	MatchID   string    `json:"match_id"`
	RaterID   string    `json:"rater_id"`
	RateeID   string    `json:"ratee_id"`
	Score     int       `json:"score"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// UserRatingResponse is the aggregate rating for a user.
type UserRatingResponse struct {
	UserID  string  `json:"user_id"`
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// MatchRatingsResponse wraps the ratings for a match.
type MatchRatingsResponse struct {
	Ratings []RatingResponse `json:"ratings"`
}
