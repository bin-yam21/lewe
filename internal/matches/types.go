package matches

import "time"

// Match statuses.
const (
	StatusPending   = "pending"
	StatusAccepted  = "accepted"
	StatusCompleted = "completed"
	StatusDeclined  = "declined"
	StatusCancelled = "cancelled"
)

// ExchangeRequest is the payload for choosing how the items will be handed over.
type ExchangeRequest struct {
	Method  string  `json:"method"`
	Details *string `json:"details"`
}

// ItemSummary is the slice of an item shown inside a match.
type ItemSummary struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Category       string   `json:"category"`
	Condition      string   `json:"condition"`
	EstimatedValue *int     `json:"estimated_value,omitempty"`
	ImageURLs      []string `json:"image_urls"`
}

// UserSummary is the slice of a user shown inside a match.
type UserSummary struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
}

// Exchange describes the agreed hand-over.
type Exchange struct {
	Method        string  `json:"method"`
	Details       *string `json:"details,omitempty"`
	ProposedByYou bool    `json:"proposed_by_you"`
}

// MatchResponse is a match as seen by one of its two participants.
type MatchResponse struct {
	ID            string      `json:"id"`
	Status        string      `json:"status"`
	Score         float64     `json:"score"`
	YourItem      ItemSummary `json:"your_item"`
	TheirItem     ItemSummary `json:"their_item"`
	OtherUser     UserSummary `json:"other_user"`
	YouAccepted   bool        `json:"you_accepted"`
	TheyAccepted  bool        `json:"they_accepted"`
	Exchange      *Exchange   `json:"exchange"`
	YouCompleted  bool        `json:"you_completed"`
	TheyCompleted bool        `json:"they_completed"`
	YouRated      bool        `json:"you_rated"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
}
