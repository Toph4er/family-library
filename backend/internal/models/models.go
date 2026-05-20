package models

import "time"

// User represents an admin user
//
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	DisplayName  string    `json:"display_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Book represents a book in the collection
//
type Book struct {
	ID                  int64     `json:"id"`
	ISBN                *string   `json:"isbn,omitempty"`
	Title               string    `json:"title"`
	Subtitle            *string   `json:"subtitle,omitempty"`
	Authors             *string   `json:"authors,omitempty"`       // JSON array
	Illustrators        *string   `json:"illustrators,omitempty"`  // JSON array
	Publisher           *string   `json:"publisher,omitempty"`
	PublicationYear     *int      `json:"publication_year,omitempty"`
	PageCount           *int      `json:"page_count,omitempty"`
	BookType            *string   `json:"book_type,omitempty"`
	ReadingLevels       *string   `json:"reading_levels,omitempty"`  // JSON array
	Genres              *string   `json:"genres,omitempty"`          // JSON array
	Themes              *string   `json:"themes,omitempty"`          // JSON array
	Awards              *string   `json:"awards,omitempty"`          // JSON array
	GiftFrom            *string   `json:"gift_from,omitempty"`
	GiftRelationship    *string   `json:"gift_relationship,omitempty"`
	DateReceived        *string   `json:"date_received,omitempty"`
	Condition           *string   `json:"condition,omitempty"`
	Location            *string   `json:"location,omitempty"`
	Notes               *string   `json:"notes,omitempty"`
	ChildRating         *int      `json:"child_rating,omitempty"`
	ReadCount           int       `json:"read_count"`
	LastReadDate        *string   `json:"last_read_date,omitempty"`
	CoverImageURL       *string   `json:"cover_image_url,omitempty"`
	CoverSource         *string   `json:"cover_source,omitempty"`
	GuestVisibleFields  string    `json:"-"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

// WishlistItem represents a book on the wishlist
//
type WishlistItem struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Author         *string    `json:"author,omitempty"`
	ISBN           *string    `json:"isbn,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	Priority       int        `json:"priority"`
	AmazonURL      *string    `json:"amazon_url,omitempty"`
	ThriftbooksURL *string    `json:"thriftbooks_url,omitempty"`
	OtherURLs      *string    `json:"other_urls,omitempty"`  // JSON array
	CoverImageURL  *string    `json:"cover_image_url,omitempty"`
	RequestedBy    *string    `json:"requested_by,omitempty"`
	RequestedAt    string   `json:"requested_at"`
	Fulfilled      bool     `json:"fulfilled"`
	FulfilledAt    *string  `json:"fulfilled_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// Setting represents an app setting
//
type Setting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description *string   `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LoginRequest represents a login request body
//
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// GuestLoginRequest represents a guest login request body
//
type GuestLoginRequest struct {
	Password string `json:"password" validate:"required"`
}

// CreateBookRequest represents a book creation request body
//
type CreateBookRequest struct {
	ISBN             *string `json:"isbn"`
	Title            string  `json:"title" validate:"required"`
	Subtitle         *string `json:"subtitle"`
	Authors          *string `json:"authors"`
	Illustrators     *string `json:"illustrators"`
	Publisher        *string `json:"publisher"`
	PublicationYear  *int    `json:"publication_year"`
	PageCount        *int    `json:"page_count"`
	BookType         *string `json:"book_type"`
	ReadingLevels    *string `json:"reading_levels"`
	Genres           *string `json:"genres"`
	Themes           *string `json:"themes"`
	Awards           *string `json:"awards"`
	GiftFrom         *string `json:"gift_from"`
	GiftRelationship *string `json:"gift_relationship"`
	DateReceived     *string `json:"date_received"`
	Condition        *string `json:"condition"`
	Location         *string `json:"location"`
	Notes            *string `json:"notes"`
	ChildRating      *int    `json:"child_rating"`
}

// UpdateBookRequest represents a book update request body
//
type UpdateBookRequest struct {
	ISBN             *string `json:"isbn"`
	Title            *string `json:"title"`
	Subtitle         *string `json:"subtitle"`
	Authors          *string `json:"authors"`
	Illustrators     *string `json:"illustrators"`
	Publisher        *string `json:"publisher"`
	PublicationYear  *int    `json:"publication_year"`
	PageCount        *int    `json:"page_count"`
	BookType         *string `json:"book_type"`
	ReadingLevels    *string `json:"reading_levels"`
	Genres           *string `json:"genres"`
	Themes           *string `json:"themes"`
	Awards           *string `json:"awards"`
	GiftFrom         *string `json:"gift_from"`
	GiftRelationship *string `json:"gift_relationship"`
	DateReceived     *string `json:"date_received"`
	Condition        *string `json:"condition"`
	Location         *string `json:"location"`
	Notes            *string `json:"notes"`
	ChildRating      *int    `json:"child_rating"`
	ReadCount        *int    `json:"read_count"`
	LastReadDate     *string `json:"last_read_date"`
	CoverImageURL    *string `json:"cover_image_url"`
	CoverSource      *string `json:"cover_source"`
}

// CreateWishlistItemRequest represents a wishlist item creation request
//
type CreateWishlistItemRequest struct {
	Title          string  `json:"title" validate:"required"`
	Author         *string `json:"author"`
	ISBN           *string `json:"isbn"`
	Reason         *string `json:"reason"`
	Priority       *int    `json:"priority"`
	AmazonURL      *string `json:"amazon_url"`
	ThriftbooksURL *string `json:"thriftbooks_url"`
	OtherURLs      *string `json:"other_urls"`
	Notes          *string `json:"notes"`
}

// UpdateWishlistItemRequest represents a wishlist item update request
//
type UpdateWishlistItemRequest struct {
	Title          *string `json:"title"`
	Author         *string `json:"author"`
	ISBN           *string `json:"isbn"`
	Reason         *string `json:"reason"`
	Priority       *int    `json:"priority"`
	AmazonURL      *string `json:"amazon_url"`
	ThriftbooksURL *string `json:"thriftbooks_url"`
	OtherURLs      *string `json:"other_urls"`
	CoverImageURL  *string `json:"cover_image_url"`
	RequestedBy    *string `json:"requested_by"`
	Notes          *string `json:"notes"`
}

// APIResponse is a generic API response wrapper
//
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginatedResponse wraps paginated data
//
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}
