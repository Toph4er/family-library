// Package pages provides template rendering for all page handlers.
// Each page gets its own file, and shared infrastructure lives in shared.go.
package pages

import (
	"html/template"

	"github.com/Toph4er/family-library/internal/models"
)

// PaginationContext holds pagination data for list pages.
type PaginationContext struct {
	Page            int
	PerPage         int
	TotalPages      int
	PaginationStart int
	PaginationEnd   int
}

// BookListContext holds data for the books listing page.
type BookListContext struct {
	Books        []models.Book
	CurrentQuery string
	TotalResults int
}

// BookDetailContext holds data for the book detail page.
type BookDetailContext struct {
	Book *models.Book
}

// BookFormContext holds data for the add/edit book form page.
type BookFormContext struct {
	BookID            int64
	IsEdit            bool
	BookTitle         string
	CancelURL         string
	ActionURL         string
	Title             string
	Subtitle          string
	Authors           string
	Illustrators      string
	ISBN              string
	Publisher         string
	PublicationYear   string
	PageCount         string
	BookType          string
	Condition         string
	Genres            string
	Themes            string
	Awards            string
	ReadingLevels     string
	DeweyDecimalClass string
	Language          string
	Series            string
	AgeRange          string
	SubjectPlaces     string
	SubjectPeople     string
	SubjectTimes      string
	Description       string
	GiftFrom          string
	GiftRelationship  string
	DateReceived      string
	Location          string
	CoverImageURL     string
	Notes             string
	ChildRating       int
	Quantity          int
}

// WishlistListContext holds data for the wishlist listing page.
type WishlistListContext struct {
	Items []models.WishlistItem
}

// WishlistFormContext holds data for the add/edit wishlist item form page.
type WishlistFormContext struct {
	IsWishlistEdit        bool
	ItemTitle             string
	WishlistCancelURL     string
	WishlistActionURL     string
	WishlistTitle         string
	Author                string
	WishlistISBN          string
	Reason                string
	Priority              int
	AmazonURL             string
	ThriftbooksURL        string
	WishlistCoverImageURL string
	WishlistNotes         string
}

// FamilyMembersContext holds family member data (shared by settings and reading-log pages).
type FamilyMembersContext struct {
	FamilyMembers []models.FamilyMember
}

// SettingsContext holds data for the settings page.
type SettingsContext struct {
	Settings               map[string]string
	Users                  []map[string]interface{}
	DefaultGuestVisibility map[string]bool
}

// ReadingLogContext holds data for the reading log page.
type ReadingLogContext struct {
	ReadingLogs []models.ReadingLog
	RecentBooks interface{} // []bookSelect
}

// StatCard represents a top-of-page stat card (big number).
type StatCard struct {
	Icon  template.HTML
	Value string
	Label string
	Link  string
}

// SectionCard represents a side-by-side info panel.
type SectionCard struct {
	Icon  template.HTML
	Title string
	Rows  []SectionRow
	Link  string
}

// SectionRow is a single label/value pair inside a SectionCard.
type SectionRow struct {
	Label string
	Value string
	Link  string
}

// ActivityEntry is a single line in the recent activity panel.
type ActivityEntry struct {
	Text string
	Time string
	Link string
}

// DashboardContext holds data for the dashboard page.
type DashboardContext struct {
	StatCards         []StatCard
	Sections          []SectionCard
	CollectionStats   []SectionRow
	Activity          []ActivityEntry
	ReaderBreakdown   []ReaderBreakdownRow
	GenreBreakdown    []GenreBreakdownRow
	BookTypeBreakdown []BookTypeBreakdownRow
}

// ReaderBreakdownRow is a single row in the "Reading by Reader" panel.
type ReaderBreakdownRow struct {
	Reader string
	Count  int
	Width  string // CSS width percentage
}

// GenreBreakdownRow is a single row in the "Genre Breakdown" panel.
type GenreBreakdownRow struct {
	Genre string
	Count int
	Width string // CSS width percentage
}

// BookTypeBreakdownRow is a single row in the "Books by Book Type" panel.
type BookTypeBreakdownRow struct {
	BookType string
	Count    int
	Width    string // CSS width percentage
}

// PageContext is the composite context used by all page handlers.
// It embeds smaller context structs so template access via {{.FieldName}}
// continues to work transparently.
type PageContext struct {
	BaseContext
	PaginationContext
	BookListContext
	BookDetailContext
	BookFormContext
	WishlistListContext
	WishlistFormContext
	FamilyMembersContext
	SettingsContext
	ReadingLogContext
	DashboardContext
}

// pageContext is an alias for backwards compatibility within the package.
type pageContext = PageContext

// wishlistColumns lists the columns to SELECT for wishlist items.
const wishlistColumns = `
	id, title, author, isbn, reason, priority,
	amazon_url, thriftbooks_url, other_urls,
	cover_image_url, requested_by, requested_at,
	fulfilled, fulfilled_at, notes
`
