package items

import "time"

// --- Request DTOs ---

// WantInput represents a want entry when creating/updating an item.
type WantInput struct {
	Category    string  `json:"category"`
	Description *string `json:"description,omitempty"`
}

// CreateItemRequest is the payload for creating a new item listing.
type CreateItemRequest struct {
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Category       string      `json:"category"`
	Condition      string      `json:"condition"`
	ExchangeMethod string      `json:"exchange_method"`
	Images         []string    `json:"images"`
	Location       *string     `json:"location,omitempty"`
	Wants          []WantInput `json:"wants"`
	// Private items never appear in the browse feed. They exist so someone can
	// offer a thing directly without publishing it to everyone first.
	Private bool `json:"private,omitempty"`
}

// UpdateItemRequest is the payload for updating an existing item.
type UpdateItemRequest struct {
	Title          *string      `json:"title,omitempty"`
	Description    *string      `json:"description,omitempty"`
	Category       *string      `json:"category,omitempty"`
	Condition      *string      `json:"condition,omitempty"`
	ExchangeMethod *string      `json:"exchange_method,omitempty"`
	Images         *[]string    `json:"images,omitempty"`
	Location       *string      `json:"location,omitempty"`
	Wants          *[]WantInput `json:"wants,omitempty"`
}

// ListItemsFilter contains query parameters for listing items.
type ListItemsFilter struct {
	Category string
	Status   string
	UserID   string
	Query    string
	Page     int
	PerPage  int
	// AllowArchived permits filtering by (and listing) archived items.
	// Set only when a user is listing their own items.
	AllowArchived bool
}

// --- Response DTOs ---

// WantResponse is the public-facing representation of a want.
type WantResponse struct {
	ID          string  `json:"id"`
	Category    string  `json:"category"`
	Description *string `json:"description,omitempty"`
}

// ItemResponse is the public-facing representation of an item listing.
type ItemResponse struct {
	ID             string         `json:"id"`
	UserID         string         `json:"user_id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Category       string         `json:"category"`
	Condition      string         `json:"condition"`
	ExchangeMethod string         `json:"exchange_method"`
	Images         []string       `json:"images"`
	Location       *string        `json:"location,omitempty"`
	Status         string         `json:"status"`
	Wants          []WantResponse `json:"wants"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ItemListResponse is a paginated list of items.
type ItemListResponse struct {
	Items   []ItemResponse `json:"items"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}
