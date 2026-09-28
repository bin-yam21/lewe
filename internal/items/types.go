package items

import "time"

// Item statuses.
const (
	StatusAvailable = "available"
	StatusReserved  = "reserved"
	StatusExchanged = "exchanged"
	StatusWithdrawn = "withdrawn"
)

// ItemRequest is the payload for creating or replacing an item listing.
type ItemRequest struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Category       string   `json:"category"`
	Condition      string   `json:"condition"`
	EstimatedValue *int     `json:"estimated_value"`
	Location       *string  `json:"location"`
	ImageURLs      []string `json:"image_urls"`
}

// ItemResponse is the public representation of an item listing.
type ItemResponse struct {
	ID             string    `json:"id"`
	OwnerID        string    `json:"owner_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Condition      string    `json:"condition"`
	EstimatedValue *int      `json:"estimated_value,omitempty"`
	Location       *string   `json:"location,omitempty"`
	ImageURLs      []string  `json:"image_urls"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
