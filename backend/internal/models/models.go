package models

import "encoding/json"

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
	Quantity            int       `json:"quantity"`
	ReadCount           int       `json:"read_count"`
	LastReadDate        *string   `json:"last_read_date,omitempty"`
	CoverImageURL       *string   `json:"cover_image_url,omitempty"`
	CoverSource         *string   `json:"cover_source,omitempty"`
	DeweyDecimalClass   *string   `json:"dewey_decimal_class,omitempty"`
	Description         *string   `json:"description,omitempty"`
	Language            *string   `json:"language,omitempty"`
	SubjectPlaces       *string   `json:"subject_places,omitempty"`  // JSON array
	SubjectPeople       *string   `json:"subject_people,omitempty"`  // JSON array
	SubjectTimes        *string   `json:"subject_times,omitempty"`   // JSON array
	AgeRange            *string   `json:"age_range,omitempty"`
	Series              *string   `json:"series,omitempty"`
	GuestVisibleFields  string    `json:"-"`
	CreatedAt           string    `json:"created_at"`
	UpdatedAt           string    `json:"updated_at"`
}

// FilterForGuest nils out fields that are not visible to guests, based on the
// GuestVisibleFields JSON blob stored per-book.  Only pointer fields are
// affected — if a field is marked false, its pointer is set to nil so it
// omits from JSON output (omitempty) or appears empty in templates.
func (b *Book) FilterForGuest() {
	if b.GuestVisibleFields == "" {
		return
	}
	var visibility map[string]bool
	if err := json.Unmarshal([]byte(b.GuestVisibleFields), &visibility); err != nil {
		return // malformed JSON — leave as-is
	}
	// Default: fields not listed are visible.
	if !visibility["isbn"] {
		b.ISBN = nil
	}
	if !visibility["subtitle"] {
		b.Subtitle = nil
	}
	if !visibility["authors"] {
		b.Authors = nil
	}
	if !visibility["illustrators"] {
		b.Illustrators = nil
	}
	if !visibility["publisher"] {
		b.Publisher = nil
	}
	if !visibility["publication_year"] {
		b.PublicationYear = nil
	}
	if !visibility["page_count"] {
		b.PageCount = nil
	}
	if !visibility["book_type"] {
		b.BookType = nil
	}
	if !visibility["reading_levels"] {
		b.ReadingLevels = nil
	}
	if !visibility["genres"] {
		b.Genres = nil
	}
	if !visibility["themes"] {
		b.Themes = nil
	}
	if !visibility["awards"] {
		b.Awards = nil
	}
	if !visibility["gift_from"] {
		b.GiftFrom = nil
	}
	if !visibility["gift_relationship"] {
		b.GiftRelationship = nil
	}
	if !visibility["date_received"] {
		b.DateReceived = nil
	}
	if !visibility["condition"] {
		b.Condition = nil
	}
	if !visibility["location"] {
		b.Location = nil
	}
	if !visibility["notes"] {
		b.Notes = nil
	}
	if !visibility["child_rating"] {
		b.ChildRating = nil
	}
	if !visibility["last_read_date"] {
		b.LastReadDate = nil
	}
	if !visibility["cover_image_url"] {
		b.CoverImageURL = nil
	}
	if !visibility["cover_source"] {
		b.CoverSource = nil
	}
}

// WishlistItem represents a book on the wishlist
//
type WishlistItem struct {
	ID             int64    `json:"id"`
	Title          string   `json:"title"`
	Author         *string  `json:"author,omitempty"`
	ISBN           *string  `json:"isbn,omitempty"`
	Reason         *string  `json:"reason,omitempty"`
	Priority       int      `json:"priority"`
	AmazonURL      *string  `json:"amazon_url,omitempty"`
	ThriftbooksURL *string  `json:"thriftbooks_url,omitempty"`
	OtherURLs      *string  `json:"other_urls,omitempty"`  // JSON array
	CoverImageURL  *string  `json:"cover_image_url,omitempty"`
	RequestedBy    *string  `json:"requested_by,omitempty"`
	RequestedAt    string   `json:"requested_at"`
	Fulfilled      bool     `json:"fulfilled"`
	FulfilledAt    *string  `json:"fulfilled_at,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	// IsAdmin is set by page handlers for template conditionals. It is not
	// serialized to JSON (json:"-" omitempty).
	IsAdmin bool `json:"-"`
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
	AgeRange         *string `json:"age_range"`
	Series           *string `json:"series"`
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
	Quantity         *int    `json:"quantity"`
	ReadCount        *int    `json:"read_count"`
	LastReadDate     *string `json:"last_read_date"`
	CoverImageURL    *string `json:"cover_image_url"`
	CoverSource      *string `json:"cover_source"`
	AgeRange         *string `json:"age_range"`
	Series           *string `json:"series"`
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

// FamilyMember represents a family member
//
type FamilyMember struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Relation  string `json:"relation"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateFamilyMemberRequest represents a family member creation request
//
type CreateFamilyMemberRequest struct {
	Name     string `json:"name" validate:"required"`
	Relation string `json:"relation" validate:"required"`
}

// UpdateFamilyMemberRequest represents a family member update request
//
type UpdateFamilyMemberRequest struct {
	Name     *string `json:"name"`
	Relation *string `json:"relation"`
}

// ReadingLog represents a reading session entry
//
type ReadingLog struct {
	ID         int64   `json:"id"`
	BookID     int64   `json:"book_id"`
	BookTitle  string  `json:"book_title"`
	StartPage  *int    `json:"start_page,omitempty"`
	EndPage    *int    `json:"end_page,omitempty"`
	TotalPages int     `json:"total_pages"`
	EntireBook bool    `json:"entire_book"`
	ReadAt     string  `json:"read_at"`
	ReaderName string  `json:"reader_name"`
	Notes      *string `json:"notes,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// CreateReadingLogRequest represents a reading log creation request
//
type CreateReadingLogRequest struct {
	BookID     int64   `json:"book_id" validate:"required"`
	StartPage  *int    `json:"start_page"`
	EndPage    *int    `json:"end_page"`
	EntireBook bool    `json:"entire_book"`
	ReadAt     string  `json:"read_at"`
	ReaderName string  `json:"reader_name" validate:"required"`
	Notes      *string `json:"notes"`
}

// UpdateReadingLogRequest represents a reading log update request
//
type UpdateReadingLogRequest struct {
	StartPage  *int    `json:"start_page"`
	EndPage    *int    `json:"end_page"`
	EntireBook *bool   `json:"entire_book"`
	ReadAt     *string `json:"read_at"`
	ReaderName *string `json:"reader_name"`
	Notes      *string `json:"notes"`
}
