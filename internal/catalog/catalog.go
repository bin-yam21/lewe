// Package catalog defines the fixed vocabularies shared by items and wants.
package catalog

// Categories are the item categories supported by the marketplace. Matching
// requires an exact category match, so a closed list keeps listings and wants comparable.
var Categories = []string{
	"electronics",
	"books",
	"clothing",
	"furniture",
	"home",
	"kitchen",
	"sports",
	"toys",
	"tools",
	"music",
	"art",
	"baby",
	"vehicles",
	"other",
}

// Conditions are item conditions ordered from best to worst.
var Conditions = []string{"new", "like_new", "good", "fair", "poor"}

// ExchangeMethods are the ways two users can hand over their items.
var ExchangeMethods = []string{"meetup", "shipping", "dropoff"}
