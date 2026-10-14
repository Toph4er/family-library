package models

import "encoding/json"

// Book represents a book in the collection
type Book struct {
	ID                 int64   `json:"id" db:"id"`
	ISBN               *string `json:"isbn,omitempty" db:"isbn"`
	Title              string  `json:"title" db:"title"`
	Subtitle           *string `json:"subtitle,omitempty" db:"subtitle"`
	Authors            *string `json:"authors,omitempty" db:"authors"`           // JSON array
	Illustrators       *string `json:"illustrators,omitempty" db:"illustrators"` // JSON array
	Publisher          *string `json:"publisher,omitempty" db:"publisher"`
	PublicationYear    *int    `json:"publication_year,omitempty" db:"publication_year"`
	PageCount          *int    `json:"page_count,omitempty" db:"page_count"`
	BookType           *string `json:"book_type,omitempty" db:"book_type"`
	ReadingLevels      *string `json:"reading_levels,omitempty" db:"reading_levels"` // JSON array
	Genres             *string `json:"genres,omitempty" db:"genres"`                 // JSON array
	Themes             *string `json:"themes,omitempty" db:"themes"`                 // JSON array
	Awards             *string `json:"awards,omitempty" db:"awards"`                 // JSON array
	GiftFrom           *string `json:"gift_from,omitempty" db:"gift_from"`
	GiftRelationship   *string `json:"gift_relationship,omitempty" db:"gift_relationship"`
	DateReceived       *string `json:"date_received,omitempty" db:"date_received"`
	Condition          *string `json:"condition,omitempty" db:"condition"`
	Location           *string `json:"location,omitempty" db:"location"`
	Source             string  `json:"source" db:"source"`
	Notes              *string `json:"notes,omitempty" db:"notes"`
	ChildRating        *int    `json:"child_rating,omitempty" db:"child_rating"`
	Quantity           int     `json:"quantity" db:"quantity"`
	ReadCount          int     `json:"read_count" db:"read_count"`
	LastReadDate       *string `json:"last_read_date,omitempty" db:"last_read_date"`
	CoverImageURL      *string `json:"cover_image_url,omitempty" db:"cover_image_url"`
	CoverSource        *string `json:"cover_source,omitempty" db:"cover_source"`
	DeweyDecimalClass  *string `json:"dewey_decimal_class,omitempty" db:"dewey_decimal_class"`
	Description        *string `json:"description,omitempty" db:"description"`
	Language           *string `json:"language,omitempty" db:"language"`
	SubjectPlaces      *string `json:"subject_places,omitempty" db:"subject_places"` // JSON array
	SubjectPeople      *string `json:"subject_people,omitempty" db:"subject_people"` // JSON array
	SubjectTimes       *string `json:"subject_times,omitempty" db:"subject_times"`   // JSON array
	AgeRange           *string `json:"age_range,omitempty" db:"age_range"`
	Series             *string `json:"series,omitempty" db:"series"`
	GuestVisibleFields string  `json:"-" db:"guest_visible_fields"`
	CreatedAt          string  `json:"created_at" db:"created_at"`
	UpdatedAt          string  `json:"updated_at" db:"updated_at"`
}

// ApplyVisibility nils out the pointer fields the admin has marked
// non-visible to guests. A field is shown only when v explicitly maps it to
// true; an absent key or false value both hide the field (matching the
// settings UI, where unchecked fields are hidden). Passing a nil map applies
// no restriction. Only pointer fields are affected — non-pointer identity
// fields (Title, Quantity, ReadCount) cannot be nulled and are left as-is.
func (b *Book) ApplyVisibility(v map[string]bool) {
	if v == nil {
		return
	}
	if !v["isbn"] {
		b.ISBN = nil
	}
	if !v["subtitle"] {
		b.Subtitle = nil
	}
	if !v["authors"] {
		b.Authors = nil
	}
	if !v["illustrators"] {
		b.Illustrators = nil
	}
	if !v["publisher"] {
		b.Publisher = nil
	}
	if !v["publication_year"] {
		b.PublicationYear = nil
	}
	if !v["page_count"] {
		b.PageCount = nil
	}
	if !v["book_type"] {
		b.BookType = nil
	}
	if !v["reading_levels"] {
		b.ReadingLevels = nil
	}
	if !v["genres"] {
		b.Genres = nil
	}
	if !v["themes"] {
		b.Themes = nil
	}
	if !v["awards"] {
		b.Awards = nil
	}
	if !v["gift_from"] {
		b.GiftFrom = nil
	}
	if !v["gift_relationship"] {
		b.GiftRelationship = nil
	}
	if !v["date_received"] {
		b.DateReceived = nil
	}
	if !v["condition"] {
		b.Condition = nil
	}
	if !v["location"] {
		b.Location = nil
	}
	if !v["notes"] {
		b.Notes = nil
	}
	if !v["child_rating"] {
		b.ChildRating = nil
	}
	if !v["last_read_date"] {
		b.LastReadDate = nil
	}
	if !v["cover_image_url"] {
		b.CoverImageURL = nil
	}
	if !v["cover_source"] {
		b.CoverSource = nil
	}
	if !v["dewey_decimal_class"] {
		b.DeweyDecimalClass = nil
	}
	if !v["description"] {
		b.Description = nil
	}
	if !v["language"] {
		b.Language = nil
	}
	if !v["subject_places"] {
		b.SubjectPlaces = nil
	}
	if !v["subject_people"] {
		b.SubjectPeople = nil
	}
	if !v["subject_times"] {
		b.SubjectTimes = nil
	}
	if !v["series"] {
		b.Series = nil
	}
	if !v["age_range"] {
		b.AgeRange = nil
	}
}

// FilterForGuest nils out fields that are not visible to guests, based on the
// GuestVisibleFields JSON blob stored per-book. Only pointer fields are
// affected — a field marked false (or absent) has its pointer set to nil so
// it omits from JSON output (omitempty) or appears empty in templates.
func (b *Book) FilterForGuest() {
	if b.GuestVisibleFields == "" {
		return
	}
	var visibility map[string]bool
	if err := json.Unmarshal([]byte(b.GuestVisibleFields), &visibility); err != nil {
		return // malformed JSON — leave as-is
	}
	b.ApplyVisibility(visibility)
}

// WishlistItem represents a book on the wishlist
type WishlistItem struct {
	ID             int64   `json:"id" db:"id"`
	Title          string  `json:"title" db:"title"`
	Author         *string `json:"author,omitempty" db:"author"`
	ISBN           *string `json:"isbn,omitempty" db:"isbn"`
	Reason         *string `json:"reason,omitempty" db:"reason"`
	Priority       int     `json:"priority" db:"priority"`
	AmazonURL      *string `json:"amazon_url,omitempty" db:"amazon_url"`
	ThriftbooksURL *string `json:"thriftbooks_url,omitempty" db:"thriftbooks_url"`
	OtherURLs      *string `json:"other_urls,omitempty" db:"other_urls"` // JSON array
	CoverImageURL  *string `json:"cover_image_url,omitempty" db:"cover_image_url"`
	RequestedBy    *string `json:"requested_by,omitempty" db:"requested_by"`
	RequestedAt    string  `json:"requested_at" db:"requested_at"`
	Fulfilled      bool    `json:"fulfilled" db:"fulfilled"`
	FulfilledAt    *string `json:"fulfilled_at,omitempty" db:"fulfilled_at"`
	Notes          *string `json:"notes,omitempty" db:"notes"`
	// IsAdmin is set by page handlers for template conditionals. It is not
	// serialized to JSON (json:"-" omitempty) and is not a DB column.
	IsAdmin bool `json:"-"`
}

// CreateBookRequest represents a book creation request body
type CreateBookRequest struct {
	ISBN              *string `json:"isbn"`
	Title             string  `json:"title" validate:"required"`
	Subtitle          *string `json:"subtitle"`
	Authors           *string `json:"authors"`
	Illustrators      *string `json:"illustrators"`
	Publisher         *string `json:"publisher"`
	PublicationYear   *int    `json:"publication_year"`
	PageCount         *int    `json:"page_count"`
	BookType          *string `json:"book_type"`
	ReadingLevels     *string `json:"reading_levels"`
	Genres            *string `json:"genres"`
	Themes            *string `json:"themes"`
	Awards            *string `json:"awards"`
	GiftFrom          *string `json:"gift_from"`
	GiftRelationship  *string `json:"gift_relationship"`
	DateReceived      *string `json:"date_received"`
	Condition         *string `json:"condition"`
	Location          *string `json:"location"`
	Notes             *string `json:"notes"`
	ChildRating       *int    `json:"child_rating"`
	DeweyDecimalClass *string `json:"dewey_decimal_class"`
	AgeRange          *string `json:"age_range"`
	Series            *string `json:"series"`
	Language          *string `json:"language"`
	SubjectPlaces     *string `json:"subject_places"`
	SubjectPeople     *string `json:"subject_people"`
	SubjectTimes      *string `json:"subject_times"`
}

// UpdateBookRequest represents a book update request body
type UpdateBookRequest struct {
	ISBN              *string `json:"isbn"`
	Title             *string `json:"title"`
	Subtitle          *string `json:"subtitle"`
	Authors           *string `json:"authors"`
	Illustrators      *string `json:"illustrators"`
	Publisher         *string `json:"publisher"`
	PublicationYear   *int    `json:"publication_year"`
	PageCount         *int    `json:"page_count"`
	BookType          *string `json:"book_type"`
	ReadingLevels     *string `json:"reading_levels"`
	Genres            *string `json:"genres"`
	Themes            *string `json:"themes"`
	Awards            *string `json:"awards"`
	GiftFrom          *string `json:"gift_from"`
	GiftRelationship  *string `json:"gift_relationship"`
	DateReceived      *string `json:"date_received"`
	Condition         *string `json:"condition"`
	Location          *string `json:"location"`
	Notes             *string `json:"notes"`
	ChildRating       *int    `json:"child_rating"`
	Quantity          *int    `json:"quantity"`
	ReadCount         *int    `json:"read_count"`
	LastReadDate      *string `json:"last_read_date"`
	CoverImageURL     *string `json:"cover_image_url"`
	CoverSource       *string `json:"cover_source"`
	DeweyDecimalClass *string `json:"dewey_decimal_class"`
	AgeRange          *string `json:"age_range"`
	Series            *string `json:"series"`
	Language          *string `json:"language"`
	SubjectPlaces     *string `json:"subject_places"`
	SubjectPeople     *string `json:"subject_people"`
	SubjectTimes      *string `json:"subject_times"`
}

// CreateWishlistItemRequest represents a wishlist item creation request
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
type FamilyMember struct {
	ID        int64  `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	Relation  string `json:"relation" db:"relation"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// CreateFamilyMemberRequest represents a family member creation request
type CreateFamilyMemberRequest struct {
	Name     string `json:"name" validate:"required"`
	Relation string `json:"relation" validate:"required"`
}

// UpdateFamilyMemberRequest represents a family member update request
type UpdateFamilyMemberRequest struct {
	Name     *string `json:"name"`
	Relation *string `json:"relation"`
}

// ReadingLog represents a reading session entry
type ReadingLog struct {
	ID         int64   `json:"id" db:"id"`
	BookID     int64   `json:"book_id" db:"book_id"`
	BookTitle  string  `json:"book_title"` // from JOIN, not a DB column
	StartPage  *int    `json:"start_page,omitempty" db:"start_page"`
	EndPage    *int    `json:"end_page,omitempty" db:"end_page"`
	TotalPages int     `json:"total_pages" db:"total_pages"`
	EntireBook bool    `json:"entire_book" db:"entire_book"`
	ReadAt     string  `json:"read_at" db:"read_at"`
	ReaderName string  `json:"reader_name" db:"reader_name"`
	Notes      *string `json:"notes,omitempty" db:"notes"`
	CreatedAt  string  `json:"created_at" db:"created_at"`
}

// CreateReadingLogRequest represents a reading log creation request
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
type UpdateReadingLogRequest struct {
	StartPage  *int    `json:"start_page"`
	EndPage    *int    `json:"end_page"`
	EntireBook *bool   `json:"entire_book"`
	ReadAt     *string `json:"read_at"`
	ReaderName *string `json:"reader_name"`
	Notes      *string `json:"notes"`
}

// User represents a user account in the system.
type User struct {
	ID          int64   `json:"id" db:"id"`
	Username    string  `json:"username" db:"username"`
	Role        string  `json:"role" db:"role"`
	DisplayName *string `json:"display_name,omitempty" db:"display_name"`
	CreatedAt   string  `json:"created_at" db:"created_at"`
}

// CreateUserRequest represents a user creation request.
type CreateUserRequest struct {
	Username    string  `json:"username" validate:"required"`
	Password    string  `json:"password" validate:"required"`
	DisplayName *string `json:"display_name,omitempty"`
}

// UpdateUserRequest represents a partial user update request.
type UpdateUserRequest struct {
	Role        *string `json:"role"`
	DisplayName *string `json:"display_name,omitempty"`
}

// ReadingLogWithBook is a reading log entry joined with its book title.
type ReadingLogWithBook struct {
	ID         int64   `json:"id" db:"id"`
	BookID     int64   `json:"book_id" db:"book_id"`
	StartPage  *int    `json:"start_page,omitempty" db:"start_page"`
	EndPage    *int    `json:"end_page,omitempty" db:"end_page"`
	TotalPages *int    `json:"total_pages,omitempty" db:"total_pages"`
	EntireBook bool    `json:"entire_book" db:"entire_book"`
	ReadAt     string  `json:"read_at" db:"read_at"`
	ReaderName *string `json:"reader_name,omitempty" db:"reader_name"`
	Notes      *string `json:"notes,omitempty" db:"notes"`
	CreatedAt  string  `json:"created_at" db:"created_at"`
	BookTitle  string  `json:"book_title"` // from JOIN, not a DB column
}

// Setting represents a key-value application setting.
type Setting struct {
	Key   string `json:"key" db:"key"`
	Value string `json:"value" db:"value"`
}
