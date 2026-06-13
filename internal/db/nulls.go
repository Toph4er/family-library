// Package db provides database initialization and connection management.
package db

import (
	"database/sql"
	"fmt"
	"time"

	"git.rcsmaine.com/chris/library/internal/models"
)

// NullStrPtr converts a sql.NullString to *string. Returns nil if not valid.
func NullStrPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// NullIntPtr64 converts a sql.NullInt64 to *int64. Returns nil if not valid.
func NullIntPtr64(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	i := ni.Int64
	return &i
}

// NullIntPtr converts a sql.NullInt64 to *int. Returns nil if not valid.
func NullIntPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	i := int(ni.Int64)
	return &i
}

// NullTimePtr converts a sql.NullTime to *string (RFC3339). Returns nil if not valid.
func NullTimePtr(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}
	s := nt.Time.Format(time.RFC3339)
	return &s
}

// NullBoolPtr converts a sql.NullBool to *bool. Returns nil if not valid.
func NullBoolPtr(nb sql.NullBool) *bool {
	if !nb.Valid {
		return nil
	}
	b := nb.Bool
	return &b
}

// StrToNullString converts a *string to sql.NullString.
// Returns an empty NullString (Valid=false) when s is nil.
func StrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// IntToNullInt64 converts a *int to sql.NullInt64.
// Returns an empty NullInt64 (Valid=false) when i is nil.
func IntToNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// BookColumns is the full column list for the books table, used in SELECT
// queries across the handlers and repository packages.
const BookColumns = `
	id, isbn, title, subtitle, authors, illustrators,
	publisher, publication_year, page_count, book_type,
	reading_levels, genres, themes, awards,
	gift_from, gift_relationship, date_received,
	condition, location, notes,
	child_rating, quantity, read_count, last_read_date,
	cover_image_url, cover_source, dewey_decimal_class, description, language,
	subject_places, subject_people, subject_times,
	series, age_range,
	guest_visible_fields, created_at, updated_at
`

// BookScanner is implemented by both *sql.Row and *sql.Rows.
type BookScanner interface {
	Scan(dest ...interface{}) error
}

// ScanBook scans a database row into a models.Book.
func ScanBook(s BookScanner) (*models.Book, error) {
	var b models.Book
	var isbn sql.NullString
	var subtitle sql.NullString
	var authors sql.NullString
	var illustrators sql.NullString
	var publisher sql.NullString
	var pubYear sql.NullInt64
	var pageCount sql.NullInt64
	var bookType sql.NullString
	var readingLevels sql.NullString
	var genres sql.NullString
	var themes sql.NullString
	var awards sql.NullString
	var giftFrom sql.NullString
	var giftRelationship sql.NullString
	var dateReceived sql.NullString
	var condition sql.NullString
	var location sql.NullString
	var notes sql.NullString
	var childRating sql.NullInt64
	var quantity sql.NullInt64
	var readCount sql.NullInt64
	var lastReadDate sql.NullString
	var coverImageURL sql.NullString
	var coverSource sql.NullString
	var deweyDecimalClass sql.NullString
	var description sql.NullString
	var language sql.NullString
	var subjectPlaces sql.NullString
	var subjectPeople sql.NullString
	var subjectTimes sql.NullString
	var series sql.NullString
	var ageRange sql.NullString
	var guestVisibleFields sql.NullString

	err := s.Scan(
		&b.ID, &isbn, &b.Title, &subtitle, &authors, &illustrators,
		&publisher, &pubYear, &pageCount, &bookType, &readingLevels, &genres,
		&themes, &awards, &giftFrom, &giftRelationship, &dateReceived,
		&condition, &location, &notes, &childRating, &quantity, &readCount,
		&lastReadDate, &coverImageURL, &coverSource, &deweyDecimalClass,
		&description, &language, &subjectPlaces, &subjectPeople, &subjectTimes,
		&series, &ageRange, &guestVisibleFields, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning book row: %w", err)
	}

	b.ISBN = NullStrPtr(isbn)
	b.Subtitle = NullStrPtr(subtitle)
	b.Authors = NullStrPtr(authors)
	b.Illustrators = NullStrPtr(illustrators)
	b.Publisher = NullStrPtr(publisher)
	b.PublicationYear = NullIntPtr(pubYear)
	b.PageCount = NullIntPtr(pageCount)
	b.BookType = NullStrPtr(bookType)
	b.ReadingLevels = NullStrPtr(readingLevels)
	b.Genres = NullStrPtr(genres)
	b.Themes = NullStrPtr(themes)
	b.Awards = NullStrPtr(awards)
	b.GiftFrom = NullStrPtr(giftFrom)
	b.GiftRelationship = NullStrPtr(giftRelationship)
	b.DateReceived = NullStrPtr(dateReceived)
	b.Condition = NullStrPtr(condition)
	b.Location = NullStrPtr(location)
	b.Notes = NullStrPtr(notes)
	b.ChildRating = NullIntPtr(childRating)

	if quantity.Valid {
		b.Quantity = int(quantity.Int64)
	} else {
		b.Quantity = 1
	}

	b.LastReadDate = NullStrPtr(lastReadDate)
	b.CoverImageURL = NullStrPtr(coverImageURL)
	b.CoverSource = NullStrPtr(coverSource)
	b.DeweyDecimalClass = NullStrPtr(deweyDecimalClass)
	b.Description = NullStrPtr(description)
	b.Language = NullStrPtr(language)
	b.SubjectPlaces = NullStrPtr(subjectPlaces)
	b.SubjectPeople = NullStrPtr(subjectPeople)
	b.SubjectTimes = NullStrPtr(subjectTimes)
	b.Series = NullStrPtr(series)
	b.AgeRange = NullStrPtr(ageRange)

	if readCount.Valid {
		b.ReadCount = int(readCount.Int64)
	}

	if guestVisibleFields.Valid {
		b.GuestVisibleFields = guestVisibleFields.String
	}

	return &b, nil
}

// ScanFamilyMember scans a database row into a models.FamilyMember.
func ScanFamilyMember(s BookScanner) (*models.FamilyMember, error) {
	var fm models.FamilyMember
	var createdAt, updatedAt string
	err := s.Scan(&fm.ID, &fm.Name, &fm.Relation, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	fm.CreatedAt = createdAt
	fm.UpdatedAt = updatedAt
	return &fm, nil
}

// ScanWishlistItem scans a database row into a models.WishlistItem.
func ScanWishlistItem(s BookScanner) (*models.WishlistItem, error) {
	var item models.WishlistItem
	var author sql.NullString
	var isbn sql.NullString
	var reason sql.NullString
	var amazonURL sql.NullString
	var thriftbooksURL sql.NullString
	var otherURLs sql.NullString
	var coverImageURL sql.NullString
	var requestedBy sql.NullString
	var fulfilledAt sql.NullTime
	var notes sql.NullString

	err := s.Scan(
		&item.ID,
		&item.Title,
		&author,
		&isbn,
		&reason,
		&item.Priority,
		&amazonURL,
		&thriftbooksURL,
		&otherURLs,
		&coverImageURL,
		&requestedBy,
		&item.RequestedAt,
		&item.Fulfilled,
		&fulfilledAt,
		&notes,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning wishlist item row: %w", err)
	}

	item.Author = NullStrPtr(author)
	item.ISBN = NullStrPtr(isbn)
	item.Reason = NullStrPtr(reason)
	item.AmazonURL = NullStrPtr(amazonURL)
	item.ThriftbooksURL = NullStrPtr(thriftbooksURL)
	item.OtherURLs = NullStrPtr(otherURLs)
	item.CoverImageURL = NullStrPtr(coverImageURL)
	item.RequestedBy = NullStrPtr(requestedBy)
	item.FulfilledAt = NullTimePtr(fulfilledAt)
	item.Notes = NullStrPtr(notes)

	return &item, nil
}
