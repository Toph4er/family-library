package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.rcsmaine.com/chris/library/internal/auth"
	"git.rcsmaine.com/chris/library/internal/models"
	"git.rcsmaine.com/chris/library/internal/repository"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"
)

// olConfig holds Open Library API configuration loaded from environment.
var olConfig = LoadOLConfig()

// --- Open Library search response types ---

// OLSearchResult represents a single book from the OL /search.json endpoint.
type OLSearchResult struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name,omitempty"`
	FirstPublishYear *int     `json:"first_publish_year,omitempty"`
	CoverI           *int     `json:"cover_i,omitempty"`
	Subject          []string `json:"subject,omitempty"`
	Language         []string `json:"language,omitempty"`
	ISBN             []string `json:"isbn,omitempty"`
}

// OLSearchResponse is the top-level envelope from OL /search.json.
type OLSearchResponse struct {
	NumFound int              `json:"numFound"`
	Start    int              `json:"start"`
	Docs     []OLSearchResult `json:"docs"`
}

// scanner is implemented by both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

// --- Helper: scan a row into a Book struct ---

func scanBook(s scanner) (*models.Book, error) {
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
	var readCount sql.NullInt64
	var guestVisibleFields sql.NullString

	err := s.Scan(
		&b.ID,
		&isbn,
		&b.Title,
		&subtitle,
		&authors,
		&illustrators,
		&publisher,
		&pubYear,
		&pageCount,
		&bookType,
		&readingLevels,
		&genres,
		&themes,
		&awards,
		&giftFrom,
		&giftRelationship,
		&dateReceived,
		&condition,
		&location,
		&notes,
		&childRating,
		&quantity,
		&readCount,
		&lastReadDate,
		&coverImageURL,
		&coverSource,
		&deweyDecimalClass,
		&description,
		&language,
		&subjectPlaces,
		&subjectPeople,
		&subjectTimes,
		&series,
		&ageRange,
		&guestVisibleFields,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning book row: %w", err)
	}

	b.ISBN = nullStrPtr(isbn)
	b.Subtitle = nullStrPtr(subtitle)
	b.Authors = nullStrPtr(authors)
	b.Illustrators = nullStrPtr(illustrators)
	b.Publisher = nullStrPtr(publisher)
	b.PublicationYear = nullIntPtr(pubYear)
	b.PageCount = nullIntPtr(pageCount)
	b.BookType = nullStrPtr(bookType)
	b.ReadingLevels = nullStrPtr(readingLevels)
	b.Genres = nullStrPtr(genres)
	b.Themes = nullStrPtr(themes)
	b.Awards = nullStrPtr(awards)
	b.GiftFrom = nullStrPtr(giftFrom)
	b.GiftRelationship = nullStrPtr(giftRelationship)
	b.DateReceived = nullStrPtr(dateReceived)
	b.Condition = nullStrPtr(condition)
	b.Location = nullStrPtr(location)
	b.Notes = nullStrPtr(notes)
	b.ChildRating = nullIntPtr(childRating)
	// quantity defaults to 1 if NULL
	if quantity.Valid {
		b.Quantity = int(quantity.Int64)
	} else {
		b.Quantity = 1
	}
	b.LastReadDate = nullStrPtr(lastReadDate)
	b.CoverImageURL = nullStrPtr(coverImageURL)
	b.CoverSource = nullStrPtr(coverSource)
	b.DeweyDecimalClass = nullStrPtr(deweyDecimalClass)
	b.Description = nullStrPtr(description)
	b.Language = nullStrPtr(language)
	b.SubjectPlaces = nullStrPtr(subjectPlaces)
	b.SubjectPeople = nullStrPtr(subjectPeople)
	b.SubjectTimes = nullStrPtr(subjectTimes)
	b.Series = nullStrPtr(series)
	b.AgeRange = nullStrPtr(ageRange)

	// read_count defaults to 0 if NULL
	if readCount.Valid {
		b.ReadCount = int(readCount.Int64)
	}
	// guest_visible_fields defaults to empty string if NULL
	if guestVisibleFields.Valid {
		b.GuestVisibleFields = guestVisibleFields.String
	}

	return &b, nil
}

func nullStrPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullIntPtr(ni sql.NullInt64) *int {
	if ni.Valid {
		v := int(ni.Int64)
		return &v
	}
	return nil
}

// --- Helper: parse pagination params ---

func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if p := r.URL.Query().Get("per_page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			perPage = v
		}
	}
	if perPage > 100 {
		perPage = 100
	}
	return
}

const bookColumns = `
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

// ListBooksHandler returns paginated list of books
func ListBooksHandler(repo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, perPage := parsePagination(r)

		q := r.URL.Query().Get("q")

		books, total, err := repo.List(r.Context(), q, page, perPage)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filterBooksForGuest(r, books)
		PaginatedResponse(w, books, total, page, perPage)
	}
}

// SearchBooksHandler searches books with focused filters
func SearchBooksHandler(repo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, perPage := parsePagination(r)

		q := r.URL.Query().Get("q")
		if q == "" {
			JSONError(w, http.StatusBadRequest, "query parameter 'q' is required")
			return
		}

		fields := []string{"title", "authors", "isbn", "genres", "themes"}

		books, total, err := repo.Search(r.Context(), q, fields, page, perPage)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filterBooksForGuest(r, books)
		PaginatedResponse(w, books, total, page, perPage)
	}
}

// GetBookHandler returns a single book
func GetBookHandler(repo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid book ID")
			return
		}

		b, err := repo.GetByID(r.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filterBookForGuest(r, b)
		JSONResponse(w, http.StatusOK, b)
	}
}

// CreateBookHandler creates a new book
func CreateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateBookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if strings.TrimSpace(req.Title) == "" {
			JSONError(w, http.StatusBadRequest, "title is required")
			return
		}

		// Normalize ISBN: strip hyphens for consistent storage
		isbn := ""
		if req.ISBN != nil {
			isbn = strings.ReplaceAll(*req.ISBN, "-", "")
		}
		if isbn == "" {
			JSONError(w, http.StatusBadRequest, "ISBN is required")
			return
		}

		// Default cover_source to 'none'
		coverSource := "none"

		query := `
			INSERT INTO books (
				isbn, title, subtitle, authors, illustrators,
				publisher, publication_year, page_count, book_type,
				reading_levels, genres, themes, awards,
				gift_from, gift_relationship, date_received,
				condition, location, notes,
				child_rating, quantity, read_count, last_read_date, cover_image_url, cover_source, dewey_decimal_class, language,
				subject_places, subject_people, subject_times, series, age_range, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := db.Exec(query,
			ptrIfNonEmpty(isbn), req.Title, req.Subtitle, req.Authors, req.Illustrators,
			req.Publisher, req.PublicationYear, req.PageCount, req.BookType,
			req.ReadingLevels, req.Genres, req.Themes, req.Awards,
			req.GiftFrom, req.GiftRelationship, req.DateReceived,
			req.Condition, req.Location, req.Notes,
			req.ChildRating,
			nil, // last_read_date
			nil, // cover_image_url
			coverSource,
			req.DeweyDecimalClass,
			req.Language,
			req.SubjectPlaces,
			req.SubjectPeople,
			req.SubjectTimes,
			req.Series,
			req.AgeRange,
			defaultGuestVisibleFields(),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				JSONError(w, http.StatusConflict, "a book with this ISBN already exists")
				return
			}
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		// Return the created book
		row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, id)
		b, err := scanBook(row)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusCreated, b)
	}
}

// UpdateBookHandler updates an existing book
func UpdateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid book ID")
			return
		}

		var req models.UpdateBookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Require title and ISBN on updates too.
		if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
			JSONError(w, http.StatusBadRequest, "title is required")
			return
		}
		if req.ISBN != nil {
			normalized := strings.ReplaceAll(strings.TrimSpace(*req.ISBN), "-", "")
			if normalized == "" {
				JSONError(w, http.StatusBadRequest, "ISBN is required")
				return
			}
		}

		// Build dynamic UPDATE from non-nil fields.
		// Empty strings are treated as "set to NULL" so fields can be cleared.
		sets := []string{}
		args := []interface{}{}

		if req.ISBN != nil {
			normalizedISBN := strings.ReplaceAll(strings.TrimSpace(*req.ISBN), "-", "")
			sets = append(sets, "isbn = ?")
			args = append(args, ptrIfNonEmpty(normalizedISBN))
		}
		if req.Title != nil {
			sets = append(sets, "title = ?")
			args = append(args, ptrIfNonEmpty(*req.Title))
		}
		if req.Subtitle != nil {
			sets = append(sets, "subtitle = ?")
			args = append(args, ptrIfNonEmpty(*req.Subtitle))
		}
		if req.Authors != nil {
			sets = append(sets, "authors = ?")
			args = append(args, ptrIfNonEmpty(*req.Authors))
		}
		if req.Illustrators != nil {
			sets = append(sets, "illustrators = ?")
			args = append(args, ptrIfNonEmpty(*req.Illustrators))
		}
		if req.Publisher != nil {
			sets = append(sets, "publisher = ?")
			args = append(args, ptrIfNonEmpty(*req.Publisher))
		}
		if req.PublicationYear != nil {
			sets = append(sets, "publication_year = ?")
			args = append(args, req.PublicationYear)
		}
		if req.PageCount != nil {
			sets = append(sets, "page_count = ?")
			args = append(args, req.PageCount)
		}
		if req.BookType != nil {
			sets = append(sets, "book_type = ?")
			args = append(args, ptrIfNonEmpty(*req.BookType))
		}
		if req.ReadingLevels != nil {
			sets = append(sets, "reading_levels = ?")
			args = append(args, ptrIfNonEmpty(*req.ReadingLevels))
		}
		if req.Genres != nil {
			sets = append(sets, "genres = ?")
			args = append(args, ptrIfNonEmpty(*req.Genres))
		}
		if req.Themes != nil {
			sets = append(sets, "themes = ?")
			args = append(args, ptrIfNonEmpty(*req.Themes))
		}
		if req.Awards != nil {
			sets = append(sets, "awards = ?")
			args = append(args, ptrIfNonEmpty(*req.Awards))
		}
		if req.GiftFrom != nil {
			sets = append(sets, "gift_from = ?")
			args = append(args, ptrIfNonEmpty(*req.GiftFrom))
		}
		if req.GiftRelationship != nil {
			sets = append(sets, "gift_relationship = ?")
			args = append(args, ptrIfNonEmpty(*req.GiftRelationship))
		}
		if req.DateReceived != nil {
			sets = append(sets, "date_received = ?")
			args = append(args, ptrIfNonEmpty(*req.DateReceived))
		}
		if req.Condition != nil {
			sets = append(sets, "condition = ?")
			args = append(args, ptrIfNonEmpty(*req.Condition))
		}
		if req.Location != nil {
			sets = append(sets, "location = ?")
			args = append(args, ptrIfNonEmpty(*req.Location))
		}
		if req.Notes != nil {
			sets = append(sets, "notes = ?")
			args = append(args, ptrIfNonEmpty(*req.Notes))
		}
		if req.ChildRating != nil {
			sets = append(sets, "child_rating = ?")
			args = append(args, req.ChildRating)
		}
		if req.Quantity != nil {
			sets = append(sets, "quantity = ?")
			args = append(args, req.Quantity)
		}
		if req.ReadCount != nil {
			sets = append(sets, "read_count = ?")
			args = append(args, req.ReadCount)
		}
		if req.LastReadDate != nil {
			sets = append(sets, "last_read_date = ?")
			args = append(args, ptrIfNonEmpty(*req.LastReadDate))
		}
		if req.CoverImageURL != nil {
			sets = append(sets, "cover_image_url = ?")
			args = append(args, ptrIfNonEmpty(*req.CoverImageURL))
		}
		if req.CoverSource != nil {
			sets = append(sets, "cover_source = ?")
			args = append(args, ptrIfNonEmpty(*req.CoverSource))
		}
		if req.DeweyDecimalClass != nil {
			sets = append(sets, "dewey_decimal_class = ?")
			args = append(args, ptrIfNonEmpty(*req.DeweyDecimalClass))
		}
		if req.Language != nil {
			sets = append(sets, "language = ?")
			args = append(args, ptrIfNonEmpty(*req.Language))
		}
		if req.SubjectPlaces != nil {
			sets = append(sets, "subject_places = ?")
			args = append(args, ptrIfNonEmpty(*req.SubjectPlaces))
		}
		if req.SubjectPeople != nil {
			sets = append(sets, "subject_people = ?")
			args = append(args, ptrIfNonEmpty(*req.SubjectPeople))
		}
		if req.SubjectTimes != nil {
			sets = append(sets, "subject_times = ?")
			args = append(args, ptrIfNonEmpty(*req.SubjectTimes))
		}
		if req.Series != nil {
			sets = append(sets, "series = ?")
			args = append(args, ptrIfNonEmpty(*req.Series))
		}
		if req.AgeRange != nil {
			sets = append(sets, "age_range = ?")
			args = append(args, ptrIfNonEmpty(*req.AgeRange))
		}

		// Always update the timestamp
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")

		if len(sets) == 1 {
			// Only updated_at was set — nothing to do
			row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, id)
			b, err := scanBook(row)
			if errors.Is(err, sql.ErrNoRows) {
				JSONError(w, http.StatusNotFound, "book not found")
				return
			}
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			filterBookForGuest(r, b)
			JSONResponse(w, http.StatusOK, b)
			return
		}

		args = append(args, id)
		// #nosec G202 -- Column names are validated against allowlist before use
		query := "UPDATE books SET " + strings.Join(sets, ", ") + " WHERE id = ?"

		result, err := db.Exec(query, args...)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				JSONError(w, http.StatusConflict, "a book with this ISBN already exists")
				return
			}
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}

		// Return the updated book
		row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, id)
		b, err := scanBook(row)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filterBookForGuest(r, b)
		JSONResponse(w, http.StatusOK, b)
	}
}

// DeleteBookHandler deletes a book
func DeleteBookHandler(repo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid book ID")
			return
		}

		err = repo.Delete(r.Context(), id)
		if err != nil {
			if strings.Contains(err.Error(), "no rows affected") {
				JSONError(w, http.StatusNotFound, "book not found")
				return
			}
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "book deleted",
		})
	}
}

// GetTagsHandler returns distinct tag values for a given tag type.
// Supported types: genres, themes, awards, reading_levels.
// The tags column stores JSON arrays like ["Fantasy","Adventure"], so we
// extract individual values using json_each in SQLite.
func GetTagsHandler(repo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagType := r.URL.Query().Get("type")

		// Map the query parameter to the actual DB column name
		column, ok := map[string]string{
			"genres":         "genres",
			"themes":         "themes",
			"awards":         "awards",
			"reading_levels": "reading_levels",
		}[tagType]
		if !ok {
			JSONError(w, http.StatusBadRequest, "invalid tag type; must be one of: genres, themes, awards, reading_levels")
			return
		}

		tags, err := repo.GetDistinctTags(r.Context(), column)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"type": tagType,
			"tags": tags,
		})
	}
}

// buildLookupResponse builds the JSON response map from a looked-up Book.
func buildLookupResponse(book *models.Book, coverSource string) map[string]interface{} {
	resp := map[string]interface{}{
		"title":        book.Title,
		"cover_source": coverSource,
	}
	if book.Subtitle != nil {
		resp["subtitle"] = *book.Subtitle
	}
	if book.Authors != nil {
		resp["authors"] = *book.Authors
	}
	if book.Illustrators != nil {
		resp["illustrators"] = *book.Illustrators
	}
	if book.Publisher != nil {
		resp["publisher"] = *book.Publisher
	}
	if book.PublicationYear != nil {
		resp["publication_year"] = *book.PublicationYear
	}
	if book.PageCount != nil {
		resp["page_count"] = *book.PageCount
	}
	if book.BookType != nil {
		resp["book_type"] = *book.BookType
	}
	if book.ReadingLevels != nil {
		resp["reading_levels"] = *book.ReadingLevels
	}
	if book.Genres != nil {
		resp["genres"] = *book.Genres
	}
	if book.Themes != nil {
		resp["themes"] = *book.Themes
	}
	if book.Awards != nil {
		resp["awards"] = *book.Awards
	}
	if book.CoverImageURL != nil {
		resp["cover_image_url"] = *book.CoverImageURL
	}
	if book.DeweyDecimalClass != nil {
		resp["dewey_decimal_class"] = *book.DeweyDecimalClass
	}
	if book.Description != nil {
		resp["description"] = *book.Description
	}
	if book.Language != nil {
		resp["language"] = *book.Language
	}
	if book.SubjectPlaces != nil {
		resp["subject_places"] = *book.SubjectPlaces
	}
	if book.SubjectPeople != nil {
		resp["subject_people"] = *book.SubjectPeople
	}
	if book.SubjectTimes != nil {
		resp["subject_times"] = *book.SubjectTimes
	}
	if book.AgeRange != nil {
		resp["age_range"] = *book.AgeRange
	}
	if book.Series != nil {
		resp["series"] = *book.Series
	}
	return resp
}

// bookFromLookupResponse reconstructs a *models.Book from a lookup response map.
// This is the inverse of buildLookupResponse, used when reading from cache.
func bookFromLookupResponse(resp map[string]interface{}) *models.Book {
	book := &models.Book{
		Title: toString(resp["title"]),
	}
	if v, ok := resp["subtitle"].(string); ok && v != "" {
		book.Subtitle = &v
	}
	if v, ok := resp["authors"].(string); ok && v != "" {
		book.Authors = &v
	}
	if v, ok := resp["illustrators"].(string); ok && v != "" {
		book.Illustrators = &v
	}
	if v, ok := resp["publisher"].(string); ok && v != "" {
		book.Publisher = &v
	}
	if v, ok := resp["publication_year"].(float64); ok {
		y := int(v)
		book.PublicationYear = &y
	}
	if v, ok := resp["page_count"].(float64); ok {
		p := int(v)
		book.PageCount = &p
	}
	if v, ok := resp["book_type"].(string); ok && v != "" {
		book.BookType = &v
	}
	if v, ok := resp["reading_levels"].(string); ok && v != "" {
		book.ReadingLevels = &v
	}
	if v, ok := resp["genres"].(string); ok && v != "" {
		book.Genres = &v
	}
	if v, ok := resp["themes"].(string); ok && v != "" {
		book.Themes = &v
	}
	if v, ok := resp["awards"].(string); ok && v != "" {
		book.Awards = &v
	}
	if v, ok := resp["cover_image_url"].(string); ok && v != "" {
		book.CoverImageURL = &v
	}
	if v, ok := resp["dewey_decimal_class"].(string); ok && v != "" {
		book.DeweyDecimalClass = &v
	}
	if v, ok := resp["description"].(string); ok && v != "" {
		book.Description = &v
	}
	if v, ok := resp["language"].(string); ok && v != "" {
		book.Language = &v
	}
	if v, ok := resp["subject_places"].(string); ok && v != "" {
		book.SubjectPlaces = &v
	}
	if v, ok := resp["subject_people"].(string); ok && v != "" {
		book.SubjectPeople = &v
	}
	if v, ok := resp["subject_times"].(string); ok && v != "" {
		book.SubjectTimes = &v
	}
	if v, ok := resp["age_range"].(string); ok && v != "" {
		book.AgeRange = &v
	}
	if v, ok := resp["series"].(string); ok && v != "" {
		book.Series = &v
	}
	return book
}

// cachedFetchFromOpenLibrary checks the SQLite cache first, then fetches from
// Open Library on a miss or if force=true. On success it stores the result in
// the cache and purges stale entries.
func cachedFetchFromOpenLibrary(db *sql.DB, isbn string, force bool) (*models.Book, string, error) {
	// Check cache unless forced
	if !force {
		var cachedData string
		var cachedAt string
		err := db.QueryRow("SELECT data, fetched_at FROM isbn_cache WHERE isbn = ?", isbn).Scan(&cachedData, &cachedAt)
		if err == nil {
			if t, parseErr := time.Parse(time.RFC3339, cachedAt); parseErr == nil && time.Since(t) < olConfig.CacheTTL {
				var resp map[string]interface{}
				if jsonErr := json.Unmarshal([]byte(cachedData), &resp); jsonErr == nil {
					book := bookFromLookupResponse(resp)
					if coverSource, ok := resp["cover_source"].(string); ok {
						return book, coverSource, nil
					}
					return book, "open_library", nil
				}
			}
		}
	}

	// Fetch from Open Library
	book, coverSource, err := fetchFromOpenLibrary(isbn)
	if err != nil || book == nil {
		return book, coverSource, err
	}

	// Cache the result
	resp := buildLookupResponse(book, coverSource)
	dataJSON, _ := json.Marshal(resp)
	_, _ = db.Exec(
		`INSERT OR REPLACE INTO isbn_cache (isbn, data, fetched_at) VALUES (?, ?, ?)`,
		isbn, string(dataJSON), time.Now().UTC().Format(time.RFC3339),
	)
	// Purge stale cache entries to prevent unbounded growth.
	cacheHours := int(olConfig.CacheTTL.Hours())
	_, _ = db.Exec(
		`DELETE FROM isbn_cache WHERE datetime(fetched_at) < datetime('now', '-' || ? || ' hours')`,
		cacheHours,
	)

	return book, coverSource, nil
}

// LookupISBNHandler looks up book metadata by ISBN without creating a record.
// Returns the data from Open Library as JSON, with SQLite caching (24h TTL).
// Pass ?force=true to bypass the cache.
func LookupISBNHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isbn := strings.ReplaceAll(strings.TrimSpace(r.URL.Query().Get("isbn")), "-", "")
		if isbn == "" {
			JSONError(w, http.StatusBadRequest, "isbn query parameter is required")
			return
		}
		force := r.URL.Query().Get("force") == "true"

		book, coverSource, apiErr := cachedFetchFromOpenLibrary(db, isbn, force)
		if apiErr != nil {
			slog.Error("Open Library lookup failed", "isbn", isbn, "error", apiErr)
			JSONError(w, http.StatusBadGateway, fmt.Sprintf("book lookup unavailable: %v", apiErr))
			return
		}
		if book == nil {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}

		resp := buildLookupResponse(book, coverSource)
		JSONResponse(w, http.StatusOK, resp)
	}
}

// ImportISBNHandler imports a book by ISBN
func ImportISBNHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Parse ISBN from request body
		var req struct {
			ISBN string `json:"isbn"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		isbn := strings.ReplaceAll(strings.TrimSpace(req.ISBN), "-", "")
		if isbn == "" {
			JSONError(w, http.StatusBadRequest, "isbn is required")
			return
		}

		// 2. Check for duplicate ISBN
		var existingID int64
		err := db.QueryRow("SELECT id FROM books WHERE isbn = ?", isbn).Scan(&existingID)
		if err == nil {
			// Book exists — return it with 409
			row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, existingID)
			existingBook, scanErr := scanBook(row)
			if scanErr != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			JSONResponse(w, http.StatusConflict, existingBook)
			return
		}
		if err != sql.ErrNoRows {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		// 3. Fetch from Open Library (uses cache if available)
		book, coverSource, apiErr := cachedFetchFromOpenLibrary(db, isbn, false)
		if apiErr != nil {
			JSONError(w, http.StatusBadGateway, "book lookup service is unavailable")
			return
		}
		if book == nil {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}

		// 5. Insert the book
		query := `
			INSERT INTO books (
				isbn, title, subtitle, authors, illustrators,
				publisher, publication_year, page_count, book_type,
				reading_levels, genres, themes, awards,
				gift_from, gift_relationship, date_received,
				condition, location, notes,
				child_rating, quantity, read_count, cover_image_url, cover_source, dewey_decimal_class, description, language,
				subject_places, subject_people, subject_times, series, age_range, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := db.Exec(query,
			isbn,
			book.Title,
			book.Subtitle,
			book.Authors,
			book.Illustrators,
			book.Publisher,
			book.PublicationYear,
			book.PageCount,
			book.BookType,
			book.ReadingLevels,
			book.Genres,
			book.Themes,
			book.Awards,
			book.GiftFrom,
			book.GiftRelationship,
			book.DateReceived,
			book.Condition,
			book.Location,
			book.Notes,
			book.ChildRating,
			book.CoverImageURL,
			coverSource,
			book.DeweyDecimalClass,
			book.Description,
			book.Language,
			book.SubjectPlaces,
			book.SubjectPeople,
			book.SubjectTimes,
			book.Series,
			book.AgeRange,
			defaultGuestVisibleFields(),
		)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		// 6. Return the created book
		row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, id)
		b, err := scanBook(row)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusCreated, b)
	}
}

// apiHTTPClient is shared across all API fetches.
var apiHTTPClient = &http.Client{
	Timeout: olConfig.HTTPTimeout,
}

// olRateLimiter caps outgoing Open Library requests at 2 req/s (burst of 1)
// to stay safely under OL's 3 req/s policy.
var olRateLimiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(olConfig.RateLimitPerSec)), 1)

// waitOLRateLimit blocks until a token is available from the Open Library
// rate limiter. It respects context cancellation so that a client disconnect
// doesn't leave a goroutine blocked.
func waitOLRateLimit(ctx context.Context) error {
	return olRateLimiter.Wait(ctx)
}

// olRequestWithRetry performs an HTTP GET request to Open Library with:
//
//   - Token-bucket rate limiting (via olRateLimiter).
//   - Exponential backoff retry (up to maxRetries attempts).
//   - 429-aware backoff: if OL returns 429 with a Retry-After header, the
//     backoff is extended to honour the server's request.
//
// It returns the response body, the HTTP status code, and any error.
// A non-2xx status is not an error — the caller inspects statusCode.
func olRequestWithRetry(ctx context.Context, url string, maxRetries int) ([]byte, int, error) {
	var body []byte
	var lastErr error
	var lastStatus int

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait for rate limiter token (respects context cancellation).
		if err := waitOLRateLimit(ctx); err != nil {
			return nil, 0, fmt.Errorf("rate limiter wait: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) // #nosec G704 — url is an OL API endpoint, not user-controlled
		if err != nil {
			return nil, 0, fmt.Errorf("building Open Library request: %w", err)
		}
		req.Header.Set("User-Agent", olConfig.UserAgent)

		resp, err := apiHTTPClient.Do(req) // #nosec G704 — url is an OL API endpoint, not user-controlled
		if err != nil {
			lastErr = fmt.Errorf("Open Library HTTP request: %w", err)
			lastStatus = 0
			continue
		}

		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("reading Open Library response: %w", err)
			lastStatus = resp.StatusCode
			continue
		}

		lastStatus = resp.StatusCode

		if resp.StatusCode == http.StatusOK {
			return body, resp.StatusCode, nil
		}

		// 429 Too Many Requests: honour Retry-After header if present.
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryAfter > 0 {
				slog.Warn("Open Library rate limit hit, honouring Retry-After",
					"url", url, "retry_after", retryAfter.String())
				if err := sleepCtx(ctx, retryAfter); err != nil {
					return nil, resp.StatusCode, fmt.Errorf("context cancelled during 429 backoff: %w", err)
				}
			}
		}

		lastErr = fmt.Errorf("Open Library returned status %d", resp.StatusCode)
	}

	// All retries exhausted — apply exponential backoff before giving up.
	if lastStatus != http.StatusOK {
		backoff := exponentialBackoff(maxRetries, 500*time.Millisecond, 5*time.Second)
		slog.Info("Open Library retry backoff before final failure",
			"url", url, "backoff", backoff.String())
		_ = sleepCtx(context.Background(), backoff) // don't cancel on final backoff
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("Open Library request failed after %d attempts", maxRetries+1)
	}
	return body, lastStatus, lastErr
}

// parseRetryAfter parses a Retry-After header value.
// Supports both decimal seconds (RFC 7231) and HTTP-date formats.
func parseRetryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	// Try decimal seconds first (most common).
	if secs, err := strconv.Atoi(header); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	// Try HTTP-date format.
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		if delay := time.Until(t); delay > 0 {
			return delay
		}
	}
	return 0
}

// exponentialBackoff computes an exponential backoff duration with jitter.
// base is the starting delay, maxDelay caps the growth.
func exponentialBackoff(attempt int, base, maxDelay time.Duration) time.Duration {
	delay := base * (1 << uint(attempt)) // base * 2^attempt
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// sleepCtx sleeps for the given duration, but returns immediately if ctx is
// cancelled. Returns nil on success, or the context's error on cancellation.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// yearRe matches a 4-digit year at the start of a string.
var yearRe = regexp.MustCompile(`^(\d{4})`)

// AuthorWorksHandler calls Open Library's /authors/{key}/works.json to retrieve
// an author's works and extract any series membership information.
// Accepts query param: key (the author OL key, e.g. "OL12345A").
func AuthorWorksHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.URL.Query().Get("key"))
		if key == "" {
			JSONError(w, http.StatusBadRequest, "query parameter 'key' is required (author OL key, e.g. OL12345A)")
			return
		}

		u := fmt.Sprintf("%s/authors/%s/works.json", olConfig.BaseURL, url.PathEscape(key))

		body, statusCode, err := olRequestWithRetry(r.Context(), u, 2)
		if err != nil {
			slog.Error("Open Library author works failed", "key", key, "error", err)
			JSONError(w, http.StatusBadGateway, fmt.Sprintf("author works unavailable: %v", err))
			return
		}
		if statusCode != http.StatusOK {
			slog.Error("Open Library author works failed", "key", key, "status", statusCode)
			JSONError(w, http.StatusBadGateway, fmt.Sprintf("author works unavailable: status %d", statusCode))
			return
		}

		// Parse the OL works response.
		// Top-level shape: { "entries": [ { "work": { ... } }, ... ] }
		var olResp struct {
			Entries []struct {
				Work map[string]interface{} `json:"work"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(body, &olResp); err != nil {
			slog.Error("failed to parse Open Library author works response", "key", key, "error", err)
			JSONError(w, http.StatusBadGateway, "failed to parse response")
			return
		}

		// Extract works with series information.
		type WorkSeriesInfo struct {
			Key         string   `json:"key"`
			Title       string   `json:"title"`
			Series      []string `json:"series,omitempty"`
			SeriesNotes []string `json:"series_notes,omitempty"`
		}

		works := make([]WorkSeriesInfo, 0, len(olResp.Entries))
		for _, entry := range olResp.Entries {
			work := entry.Work
			if work == nil {
				continue
			}

			ws := WorkSeriesInfo{
				Key:   toString(work["key"]),
				Title: toString(work["title"]),
			}

			// series can be a single string or an array of strings.
			if raw, ok := work["series"]; ok {
				switch v := raw.(type) {
				case string:
					if v != "" {
						ws.Series = []string{v}
					}
				case []interface{}:
					for _, s := range v {
						if s, ok := s.(string); ok && s != "" {
							ws.Series = append(ws.Series, s)
						}
					}
				}
			}

			// series_notes is always an array.
			if raw, ok := work["series_notes"].([]interface{}); ok {
				for _, s := range raw {
					if s, ok := s.(string); ok && s != "" {
						ws.SeriesNotes = append(ws.SeriesNotes, s)
					}
				}
			}

			works = append(works, ws)
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"author_key":  key,
			"total_works": len(works),
			"works":       works,
		})
	}
}

// SearchOpenLibraryHandler searches Open Library's catalog via /search.json.
// Accepts query params: q, subject, language, author, page, limit.
func SearchOpenLibraryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		subject := r.URL.Query().Get("subject")
		language := r.URL.Query().Get("language")
		author := r.URL.Query().Get("author")

		// Pagination
		page := 1
		limit := 20
		if p := r.URL.Query().Get("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				limit = v
			}
		}
		if limit > 50 {
			limit = 50
		}

		// OL /search.json uses 1-based page numbers.
		params := url.Values{}
		if q != "" {
			params.Set("q", q)
		}
		if subject != "" {
			params.Set("subject", subject)
		}
		if language != "" {
			params.Set("language", language)
		}
		if author != "" {
			params.Set("author", author)
		}
		params.Set("page", strconv.Itoa(page))
		params.Set("limit", strconv.Itoa(limit))

		u := fmt.Sprintf("%s/search.json?%s", olConfig.BaseURL, params.Encode())

		body, statusCode, err := olRequestWithRetry(r.Context(), u, 2)
		if err != nil {
			slog.Error("Open Library search failed", "error", err)
			JSONError(w, http.StatusBadGateway, fmt.Sprintf("search unavailable: %v", err))
			return
		}
		if statusCode != http.StatusOK {
			slog.Error("Open Library search failed", "status", statusCode)
			JSONError(w, http.StatusBadGateway, fmt.Sprintf("search unavailable: status %d", statusCode))
			return
		}

		var searchResp OLSearchResponse
		if err := json.Unmarshal(body, &searchResp); err != nil {
			slog.Error("failed to parse Open Library search response", "error", err)
			JSONError(w, http.StatusBadGateway, "failed to parse search response")
			return
		}

		JSONResponse(w, http.StatusOK, searchResp)
	}
}

// fetchFromOpenLibrary fetches book metadata from the Open Library API.
// Returns the populated book, cover source string, and any error.
// If no results are found, returns (nil, "", nil).
func fetchFromOpenLibrary(isbn string) (*models.Book, string, error) {
	u := fmt.Sprintf("%s/isbn/%s.json", olConfig.BaseURL, url.PathEscape(isbn))
	body, statusCode, err := olRequestWithRetry(context.Background(), u, 2)
	if err != nil {
		return nil, "", fmt.Errorf("Open Library request failed after retries: %w", err)
	}

	if statusCode == http.StatusNotFound {
		return nil, "", nil
	}
	if statusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Open Library returned status %d", statusCode)
	}

	var olResp map[string]interface{}
	if err := json.Unmarshal(body, &olResp); err != nil {
		return nil, "", fmt.Errorf("parsing Open Library JSON: %w", err)
	}

	book := &models.Book{
		Title: toString(olResp["title"]),
	}

	// Subtitle
	if subtitle, ok := olResp["subtitle"].(string); ok && subtitle != "" {
		book.Subtitle = &subtitle
	}

	// Authors: array of {name: "..."} or {key: "..."}
	// If "name" is present, use it; if only "key" is present, resolve via the authors API.
	if authorsRaw, ok := olResp["authors"].([]interface{}); ok && len(authorsRaw) > 0 {
		authors := resolveOpenLibraryAuthorKeys(authorsRaw)
		if len(authors) > 0 {
			authorsJSON, _ := json.Marshal(authors)
			s := string(authorsJSON)
			book.Authors = &s
		}
	}

	// Publisher: the API returns "publishers" (plural) as an array of strings.
	if pubRaw, ok := olResp["publishers"]; ok {
		switch p := pubRaw.(type) {
		case string:
			if p != "" {
				book.Publisher = &p
			}
		case []interface{}:
			if len(p) > 0 {
				if pub, ok := p[0].(string); ok && pub != "" {
					book.Publisher = &pub
				}
			}
		}
	}

	// Publish date: extract year
	if publishDate, ok := olResp["publish_date"].(string); ok {
		if m := yearRe.FindStringSubmatch(publishDate); m != nil {
			year, _ := strconv.Atoi(m[1])
			book.PublicationYear = &year
		}
	}

	// Page count: field is "number_of_pages" (not "number_of_pages_num")
	if pagesRaw, ok := olResp["number_of_pages"].(float64); ok && pagesRaw > 0 {
		pages := int(pagesRaw)
		book.PageCount = &pages
	}

	// Subjects: map to genres
	if subjectsRaw, ok := olResp["subjects"].([]interface{}); ok && len(subjectsRaw) > 0 {
		var subjects []string
		for _, s := range subjectsRaw {
			if subject, ok := s.(string); ok && subject != "" {
				subjects = append(subjects, subject)
			}
		}
		if len(subjects) > 0 {
			subjectsJSON, _ := json.Marshal(subjects)
			s := string(subjectsJSON)
			book.Genres = &s
		}
	}

	// Subject dimensions: places, people, times — arrays of strings from OL.
	// Stored as JSON strings, same pattern as genres.
	extractStringSlice := func(field string) *string {
		if raw, ok := olResp[field].([]interface{}); ok && len(raw) > 0 {
			var vals []string
			for _, v := range raw {
				if s, ok := v.(string); ok && s != "" {
					vals = append(vals, s)
				}
			}
			if len(vals) > 0 {
				b, _ := json.Marshal(vals)
				s := string(b)
				return &s
			}
		}
		return nil
	}
	book.SubjectPlaces = extractStringSlice("subject_places")
	book.SubjectPeople = extractStringSlice("subject_people")
	book.SubjectTimes = extractStringSlice("subject_times")

	// Illustrators: array of {name: "..."} or {key: "..."} (same pattern as authors)
	if illustratorsRaw, ok := olResp["illustrators"].([]interface{}); ok && len(illustratorsRaw) > 0 {
		illustrators := resolveOpenLibraryAuthorKeys(illustratorsRaw)
		if len(illustrators) > 0 {
			illustratorsJSON, _ := json.Marshal(illustrators)
			s := string(illustratorsJSON)
			book.Illustrators = &s
		}
	}

	// Cover image: try cover_i first, then fall back to the covers array.
	// We intentionally do NOT use ISBN-based URLs (/b/isbn/{isbn}-L.jpg) because
	// they return 1x1 placeholder images when no cover actually exists.
	var coverID float64
	if ci, ok := olResp["cover_i"].(float64); ok && ci > 0 {
		coverID = ci
	} else if coversRaw, ok := olResp["covers"].([]interface{}); ok && len(coversRaw) > 0 {
		if firstCover, ok := coversRaw[0].(float64); ok && firstCover > 0 {
			coverID = firstCover
		}
	}
	if coverID > 0 {
		coverURL := fmt.Sprintf("%s/b/id/%d-L.jpg", olConfig.CoversURL, int(coverID))
		book.CoverImageURL = &coverURL
	}

	// Fallback: if no cover found via cover_i or covers array, try the work's OLID.
	// The OL response may have a "work" field with a "key" like "/works/OL12345W".
	if book.CoverImageURL == nil {
		if workRaw, ok := olResp["work"].(map[string]interface{}); ok {
			if workKey, ok := workRaw["key"].(string); ok && workKey != "" {
				// Extract OLID from "/works/OL12345W" -> "OL12345W"
				workKeyParts := strings.Split(workKey, "/")
				if len(workKeyParts) > 0 {
					olid := workKeyParts[len(workKeyParts)-1]
					coverURL := fmt.Sprintf("%s/b/olid/%s-L.jpg", olConfig.CoversURL, olid)
					book.CoverImageURL = &coverURL
				}
			}
		}
	}

	// Dewey Decimal Classification: can be a string or an array of strings.
	// Take the first value if it's an array.
	if ddcRaw, ok := olResp["dewey_decimal_class"]; ok {
		switch d := ddcRaw.(type) {
		case string:
			if d != "" {
				book.DeweyDecimalClass = &d
			}
		case []interface{}:
			if len(d) > 0 {
				if s, ok := d[0].(string); ok && s != "" {
					book.DeweyDecimalClass = &s
				}
			}
		}
	}

	// Description: OL returns a text type — either a plain string or a map
	// with "type" and "value" keys. Extract the plain string value.
	if descRaw, ok := olResp["description"]; ok {
		switch d := descRaw.(type) {
		case string:
			if d != "" {
				book.Description = &d
			}
		case map[string]interface{}:
			if v, ok := d["value"].(string); ok && v != "" {
				book.Description = &v
			}
		}
	}

	// Language: can be a string or an array of strings. Take the first one.
	if langRaw, ok := olResp["language"]; ok {
		switch l := langRaw.(type) {
		case string:
			if l != "" {
				book.Language = &l
			}
		case []interface{}:
			if len(l) > 0 {
				if s, ok := l[0].(string); ok && s != "" {
					book.Language = &s
				}
			}
		}
	}

	return book, "open_library", nil
}

// resolveOpenLibraryAuthorKeys resolves author entries from the Open Library API.
// Each entry is a map with either a "name" field or a "key" field.
// Entries with "name" are used directly. Entries with only "key" are resolved
// by fetching the author record from the Open Library authors API.
func resolveOpenLibraryAuthorKeys(raw []interface{}) []string {
	var named []string
	var keysToResolve []string

	for _, a := range raw {
		if m, ok := a.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok && name != "" {
				named = append(named, name)
			} else if key, ok := m["key"].(string); ok && key != "" {
				keysToResolve = append(keysToResolve, key)
			}
		}
	}

	// Resolve author keys in parallel (capped at 3 concurrent requests)
	if len(keysToResolve) > 0 {
		results := make(chan string, len(keysToResolve))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 3) // max 3 concurrent author resolutions
		for _, key := range keysToResolve {
			sem <- struct{}{}
			wg.Add(1)
			go func(k string) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if name := fetchOpenLibraryAuthorName(k); name != "" {
					results <- name
				} else {
					slog.Warn("failed to resolve Open Library author key", "key", k)
				}
			}(key)
		}
		wg.Wait()
		close(results)
		for name := range results {
			named = append(named, name)
		}
	}

	return named
}

// fetchOpenLibraryAuthorName fetches the name of a single author by their OL key.
func fetchOpenLibraryAuthorName(key string) string {
	u := fmt.Sprintf("%s%s.json", olConfig.BaseURL, key)
	body, statusCode, err := olRequestWithRetry(context.Background(), u, 2)
	if err != nil {
		slog.Warn("Open Library author request failed after retries", "key", key, "error", err)
		return ""
	}
	if statusCode != http.StatusOK {
		slog.Warn("Open Library author request failed", "key", key, "status", statusCode)
		return ""
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		slog.Warn("failed to parse Open Library author response", "key", key, "error", err)
		return ""
	}

	if name, ok := data["name"].(string); ok {
		return name
	}
	slog.Warn("Open Library author response missing name field", "key", key)
	return ""
}

// toString safely converts an interface{} to string.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// defaultGuestVisibleFields returns the default JSON blob for guest visibility.
// Most fields are visible except isbn, condition, location, notes, date_received, last_read_date.
func defaultGuestVisibleFields() string {
	fields := map[string]bool{
		"title":               true,
		"subtitle":            true,
		"authors":             true,
		"illustrators":        true,
		"publisher":           true,
		"publication_year":    true,
		"page_count":          true,
		"quantity":            true,
		"book_type":           true,
		"reading_levels":      true,
		"genres":              true,
		"themes":              true,
		"awards":              true,
		"gift_from":           true,
		"gift_relationship":   true,
		"child_rating":        true,
		"read_count":          true,
		"cover_image_url":     true,
		"cover_source":        true,
		"dewey_decimal_class": true,
		"description":         true,
		"language":            true,
		"subject_places":      true,
		"subject_people":      true,
		"subject_times":       true,
		"series":              true,
		"age_range":           true,
		"isbn":                false,
		"condition":           false,
		"location":            false,
		"notes":               false,
		"date_received":       false,
		"last_read_date":      false,
	}
	b, _ := json.Marshal(fields)
	return string(b)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// filterBookForGuest filters a single book for guest visibility if the current
// user is a guest.
func filterBookForGuest(r *http.Request, b *models.Book) {
	if user := auth.GetUserFromContext(r); user != nil && user.IsGuest {
		b.FilterForGuest()
	}
}

// filterBooksForGuest filters a slice of books for guest visibility if the
// current user is a guest.
func filterBooksForGuest(r *http.Request, books []models.Book) {
	user := auth.GetUserFromContext(r)
	if user == nil || !user.IsGuest {
		return
	}
	for i := range books {
		books[i].FilterForGuest()
	}
}

// HTMLCreateBookHandler handles POST /books/create (form submission from the add-book page).
func HTMLCreateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Invalid request</div>`)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Title is required</div>`)
			return
		}

		isbn := strings.ReplaceAll(strings.TrimSpace(r.FormValue("isbn")), "-", "")
		if isbn == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">ISBN is required</div>`)
			return
		}

		coverSource := "none"
		childRatingStr := r.FormValue("child_rating")
		var childRating *int
		if childRatingStr != "" {
			if v, err := strconv.Atoi(childRatingStr); err == nil {
				childRating = &v
			}
		}
		pubYearStr := r.FormValue("publication_year")
		var pubYear *int
		if pubYearStr != "" {
			if v, err := strconv.Atoi(pubYearStr); err == nil {
				pubYear = &v
			}
		}
		pageCountStr := r.FormValue("page_count")
		var pageCount *int
		if pageCountStr != "" {
			if v, err := strconv.Atoi(pageCountStr); err == nil {
				pageCount = &v
			}
		}

		query := `
			INSERT INTO books (
				isbn, title, subtitle, authors, illustrators,
				publisher, publication_year, page_count, book_type,
				reading_levels, genres, themes, awards,
				gift_from, gift_relationship, date_received,
				condition, location, notes,
				child_rating, quantity, read_count, last_read_date, cover_image_url, cover_source, dewey_decimal_class, language,
				subject_places, subject_people, subject_times, series, age_range, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := db.Exec(query,
			ptrIfNonEmpty(isbn), title,
			ptrIfNonEmpty(r.FormValue("subtitle")),
			ptrIfNonEmpty(r.FormValue("authors")),
			ptrIfNonEmpty(r.FormValue("illustrators")),
			ptrIfNonEmpty(r.FormValue("publisher")),
			pubYear, pageCount,
			ptrIfNonEmpty(r.FormValue("book_type")),
			ptrIfNonEmpty(r.FormValue("reading_levels")),
			ptrIfNonEmpty(r.FormValue("genres")),
			ptrIfNonEmpty(r.FormValue("themes")),
			ptrIfNonEmpty(r.FormValue("awards")),
			ptrIfNonEmpty(r.FormValue("gift_from")),
			ptrIfNonEmpty(r.FormValue("gift_relationship")),
			ptrIfNonEmpty(r.FormValue("date_received")),
			ptrIfNonEmpty(r.FormValue("condition")),
			ptrIfNonEmpty(r.FormValue("location")),
			ptrIfNonEmpty(r.FormValue("notes")),
			childRating,
			nil, // last_read_date
			ptrIfNonEmpty(r.FormValue("cover_image_url")),
			coverSource,
			ptrIfNonEmpty(r.FormValue("dewey_decimal_class")),
			ptrIfNonEmpty(r.FormValue("language")),
			ptrIfNonEmpty(r.FormValue("subject_places")),
			ptrIfNonEmpty(r.FormValue("subject_people")),
			ptrIfNonEmpty(r.FormValue("subject_times")),
			ptrIfNonEmpty(r.FormValue("series")),
			ptrIfNonEmpty(r.FormValue("age_range")),
			defaultGuestVisibleFields(),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusConflict)
				fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">A book with this ISBN already exists</div>`)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Failed to create book</div>`)
			return
		}

		id, _ := result.LastInsertId()
		w.Header().Set("HX-Redirect", "/books/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLUpdateBookHandler handles POST /books/{id}/update (form submission from the edit-book page).
func HTMLUpdateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Invalid book ID</div>`)
			return
		}

		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Invalid request</div>`)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Title is required</div>`)
			return
		}
		isbn := strings.ReplaceAll(strings.TrimSpace(r.FormValue("isbn")), "-", "")
		if isbn == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">ISBN is required</div>`)
			return
		}

		// Build dynamic UPDATE from form fields
		sets := []string{}
		args := []interface{}{}

		formFields := []string{
			"isbn",
			"subtitle",
			"authors",
			"illustrators",
			"publisher",
			"book_type",
			"reading_levels",
			"genres",
			"themes",
			"awards",
			"gift_from",
			"gift_relationship",
			"date_received",
			"condition",
			"location",
			"notes",
			"cover_image_url",
			"dewey_decimal_class",
			"language",
			"subject_places",
			"subject_people",
			"subject_times",
			"series",
			"age_range",
		}

		// ISBN - already validated and normalized above; add to update sets
		sets = append(sets, "isbn = ?")
		args = append(args, ptrIfNonEmpty(isbn))

		// Title - already validated above; add to update sets
		sets = append(sets, "title = ?")
		args = append(args, ptrIfNonEmpty(title))

		// String fields - always set; empty string means clear to NULL
		for _, name := range formFields {
			if name == "isbn" {
				continue // already handled
			}
			val := strings.TrimSpace(r.FormValue(name))
			sets = append(sets, name+" = ?")
			args = append(args, ptrIfNonEmpty(val))
		}

		// Integer fields - always set; empty string means clear to NULL
		if v := strings.TrimSpace(r.FormValue("publication_year")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				sets = append(sets, "publication_year = ?")
				args = append(args, &n)
			} else {
				sets = append(sets, "publication_year = ?")
				args = append(args, nil)
			}
		} else {
			sets = append(sets, "publication_year = ?")
			args = append(args, nil)
		}
		if v := strings.TrimSpace(r.FormValue("page_count")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				sets = append(sets, "page_count = ?")
				args = append(args, &n)
			} else {
				sets = append(sets, "page_count = ?")
				args = append(args, nil)
			}
		} else {
			sets = append(sets, "page_count = ?")
			args = append(args, nil)
		}
		if v := strings.TrimSpace(r.FormValue("child_rating")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				sets = append(sets, "child_rating = ?")
				args = append(args, &n)
			} else {
				sets = append(sets, "child_rating = ?")
				args = append(args, nil)
			}
		} else {
			sets = append(sets, "child_rating = ?")
			args = append(args, nil)
		}
		if v := strings.TrimSpace(r.FormValue("quantity")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 {
				sets = append(sets, "quantity = ?")
				args = append(args, &n)
			} else {
				sets = append(sets, "quantity = ?")
				args = append(args, nil)
			}
		} else {
			sets = append(sets, "quantity = ?")
			args = append(args, nil)
		}

		// Always update timestamp
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, id)

		// #nosec G202 -- Column names are hardcoded
		query := "UPDATE books SET " + strings.Join(sets, ", ") + " WHERE id = ?"
		result, err := db.Exec(query, args...)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusConflict)
				fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">A book with this ISBN already exists</div>`)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Failed to update book</div>`)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `<div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg" role="alert">Book not found</div>`)
			return
		}

		w.Header().Set("HX-Redirect", "/books/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusOK)
	}
}

// ptrIfNonEmpty returns a pointer to s if s is non-empty, nil otherwise.
func ptrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
