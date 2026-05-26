package handlers

import (
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

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

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
	var lastReadDate sql.NullString
	var coverImageURL sql.NullString
	var coverSource sql.NullString
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
		&readCount,
		&lastReadDate,
		&coverImageURL,
		&coverSource,
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
	b.LastReadDate = nullStrPtr(lastReadDate)
	b.CoverImageURL = nullStrPtr(coverImageURL)
	b.CoverSource = nullStrPtr(coverSource)

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
	child_rating, read_count, last_read_date,
	cover_image_url, cover_source, guest_visible_fields,
	created_at, updated_at
`

// ListBooksHandler returns paginated list of books
func ListBooksHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, perPage := parsePagination(r)
		offset := (page - 1) * perPage

		q := r.URL.Query().Get("q")

		// Count query
		var countQuery string
		var countArgs []interface{}

		if q != "" {
			countQuery = `SELECT COUNT(*) FROM books WHERE title LIKE ? OR authors LIKE ? OR isbn LIKE ? OR genres LIKE ? OR themes LIKE ?`
			like := "%" + q + "%"
			countArgs = []interface{}{like, like, like, like, like}
		} else {
			countQuery = `SELECT COUNT(*) FROM books`
		}

		var total int
		if err := db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		// Data query
		var dataQuery string
		var dataArgs []interface{}

		if q != "" {
			like := "%" + q + "%"
			dataQuery = `SELECT ` + bookColumns + ` FROM books WHERE title LIKE ? OR authors LIKE ? OR isbn LIKE ? OR genres LIKE ? OR themes LIKE ? ORDER BY title ASC LIMIT ? OFFSET ?`
			dataArgs = []interface{}{like, like, like, like, like, perPage, offset}
		} else {
			dataQuery = `SELECT ` + bookColumns + ` FROM books ORDER BY title ASC LIMIT ? OFFSET ?`
			dataArgs = []interface{}{perPage, offset}
		}

		rows, err := db.Query(dataQuery, dataArgs...)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			b, err := scanBook(rows)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			books = append(books, *b)
		}
		if err = rows.Err(); err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filterBooksForGuest(r, books)
		PaginatedResponse(w, books, total, page, perPage)
	}
}

// SearchBooksHandler searches books with focused filters
func SearchBooksHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, perPage := parsePagination(r)
		offset := (page - 1) * perPage

		q := r.URL.Query().Get("q")
		if q == "" {
			JSONError(w, http.StatusBadRequest, "query parameter 'q' is required")
			return
		}

		author := r.URL.Query().Get("author")
		bookType := r.URL.Query().Get("book_type")
		pageCountMin := r.URL.Query().Get("page_count_min")
		pageCountMax := r.URL.Query().Get("page_count_max")

		// Build WHERE clause dynamically
		conditions := []string{}
		args := []interface{}{}

		// Core text search
		like := "%" + q + "%"
		conditions = append(conditions, "(title LIKE ? OR authors LIKE ? OR isbn LIKE ? OR genres LIKE ? OR themes LIKE ?)")
		args = append(args, like, like, like, like, like)

		if author != "" {
			conditions = append(conditions, "authors LIKE ?")
			args = append(args, "%"+author+"%")
		}
		if bookType != "" {
			conditions = append(conditions, "book_type = ?")
			args = append(args, bookType)
		}
		if pageCountMin != "" {
			if min, err := strconv.Atoi(pageCountMin); err == nil {
				conditions = append(conditions, "page_count >= ?")
				args = append(args, min)
			}
		}
		if pageCountMax != "" {
			if max, err := strconv.Atoi(pageCountMax); err == nil {
				conditions = append(conditions, "page_count <= ?")
				args = append(args, max)
			}
		}

		where := ""
		if len(conditions) > 0 {
			where = " WHERE " + strings.Join(conditions, " AND ")
		}

		// Count
		countQuery := "SELECT COUNT(*) FROM books" + where
		var total int
		if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		// Data
		dataArgs := append(args, perPage, offset)
		dataQuery := "SELECT " + bookColumns + " FROM books" + where + " ORDER BY title ASC LIMIT ? OFFSET ?"

		rows, err := db.Query(dataQuery, dataArgs...)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			b, err := scanBook(rows)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			books = append(books, *b)
		}
		if err = rows.Err(); err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		filterBooksForGuest(r, books)
		PaginatedResponse(w, books, total, page, perPage)
	}
}

// GetBookHandler returns a single book
func GetBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid book ID")
			return
		}

		query := `SELECT ` + bookColumns + ` FROM books WHERE id = ?`
		row := db.QueryRow(query, id)

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
				child_rating, read_count, last_read_date, cover_image_url, cover_source, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)
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
func DeleteBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid book ID")
			return
		}

		result, err := db.Exec("DELETE FROM books WHERE id = ?", id)
		if err != nil {
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
func GetTagsHandler(db *sql.DB) http.HandlerFunc {
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

		// SQLite's json_each extracts each element of a JSON array as a row.
		// We get distinct, non-null values, ordered alphabetically.
		query := fmt.Sprintf(
			`SELECT DISTINCT value FROM books, json_each(books.%s) WHERE value IS NOT NULL AND value != '' ORDER BY value COLLATE NOCASE`,
			column,
		)

		rows, err := db.Query(query)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		defer rows.Close()

		tags := make([]string, 0)
		for rows.Next() {
			var tag string
			if err := rows.Scan(&tag); err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		if err = rows.Err(); err != nil {
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
	return resp
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

		// Check cache unless forced
		if !force {
			var cachedData string
			var cachedAt string
			err := db.QueryRow("SELECT data, fetched_at FROM isbn_cache WHERE isbn = ?", isbn).Scan(&cachedData, &cachedAt)
			if err == nil {
				// Check if cache is within 24h
				if t, parseErr := time.Parse(time.RFC3339, cachedAt); parseErr == nil && time.Since(t) < 24*time.Hour {
					var resp map[string]interface{}
					if jsonErr := json.Unmarshal([]byte(cachedData), &resp); jsonErr == nil {
						JSONResponse(w, http.StatusOK, resp)
						return
					}
				}
			}
		}

		// Fetch from Open Library, fall back to Google Books
		book, coverSource, apiErr := fetchFromOpenLibrary(isbn)
		if apiErr != nil {
			slog.Warn("Open Library lookup failed, trying Google Books", "isbn", isbn, "error", apiErr)
			book, coverSource, apiErr = fetchFromGoogleBooks(isbn)
			if apiErr != nil {
				slog.Error("All book lookup services failed", "isbn", isbn, "error", apiErr)
				JSONError(w, http.StatusBadGateway, fmt.Sprintf("book lookup unavailable: %v", apiErr))
				return
			}
		}
		if book == nil {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}

		resp := buildLookupResponse(book, coverSource)

		// Cache the result
		dataJSON, _ := json.Marshal(resp)
		_, _ = db.Exec(
			`INSERT OR REPLACE INTO isbn_cache (isbn, data, fetched_at) VALUES (?, ?, ?)`,
			isbn, string(dataJSON), time.Now().UTC().Format(time.RFC3339),
		)
		// Purge stale cache entries (older than 24h) to prevent unbounded growth.
		_, _ = db.Exec(
			`DELETE FROM isbn_cache WHERE datetime(fetched_at) < datetime('now', '-24 hours')`,
		)

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

		// 3. Fetch from Open Library
		book, coverSource, apiErr := fetchFromOpenLibrary(isbn)
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
				child_rating, read_count, cover_image_url, cover_source, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
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
	Timeout: 10 * time.Second,
}

// yearRe matches a 4-digit year at the start of a string.
var yearRe = regexp.MustCompile(`^(\d{4})`)

// fetchFromOpenLibrary fetches book metadata from the Open Library API.
// Returns the populated book, cover source string, and any error.
// If no results are found, returns (nil, "", nil).
func fetchFromOpenLibrary(isbn string) (*models.Book, string, error) {
	u := fmt.Sprintf("https://openlibrary.org/isbn/%s.json", url.PathEscape(isbn))
	// #nosec G704 -- URL domain is hardcoded to openlibrary.org (trusted external API);
	// isbn is sanitized with url.PathEscape. This is a client-side lookup, not a redirect.
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("building Open Library request: %w", err)
	}

	// #nosec G704 -- URL is constructed from hardcoded openlibrary.org base; isbn is sanitized
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Open Library HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading Open Library response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Open Library returned status %d", resp.StatusCode)
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
		coverURL := fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", int(coverID))
		book.CoverImageURL = &coverURL
	}

	return book, "open_library", nil
}

// fetchFromGoogleBooks fetches book metadata from the Google Books API.
func fetchFromGoogleBooks(isbn string) (*models.Book, string, error) {
	u := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s&maxResults=1", url.QueryEscape(isbn))
	// #nosec G704 -- URL domain is hardcoded to googleapis.com (trusted external API);
	// isbn is sanitized with url.QueryEscape. This is a client-side lookup, not a redirect.
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("building Google Books request: %w", err)
	}

	// #nosec G704 -- URL is constructed from hardcoded googleapis.com base; isbn is sanitized
	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Google Books HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading Google Books response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Google Books returned status %d", resp.StatusCode)
	}

	var gbResp map[string]interface{}
	if err := json.Unmarshal(body, &gbResp); err != nil {
		return nil, "", fmt.Errorf("parsing Google Books JSON: %w", err)
	}

	items, ok := gbResp["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, "", nil
	}

	volumeInfo, ok := items[0].(map[string]interface{})["volumeInfo"].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("unexpected Google Books response structure")
	}

	book := &models.Book{
		Title: toString(volumeInfo["title"]),
	}

	if subtitle, ok := volumeInfo["subtitle"].(string); ok && subtitle != "" {
		book.Subtitle = &subtitle
	}

	// Authors
	if authorsRaw, ok := volumeInfo["authors"].([]interface{}); ok && len(authorsRaw) > 0 {
		var authors []string
		for _, a := range authorsRaw {
			if name, ok := a.(string); ok && name != "" {
				authors = append(authors, name)
			}
		}
		if len(authors) > 0 {
			authorsJSON, _ := json.Marshal(authors)
			s := string(authorsJSON)
			book.Authors = &s
		}
	}

	// Publisher
	if pub, ok := volumeInfo["publisher"].(string); ok && pub != "" {
		book.Publisher = &pub
	}

	// Publication year
	if pubDate, ok := volumeInfo["publishedDate"].(string); ok && pubDate != "" {
		if m := yearRe.FindStringSubmatch(pubDate); m != nil {
			year, _ := strconv.Atoi(m[1])
			book.PublicationYear = &year
		}
	}

	// Page count
	if pagesRaw, ok := volumeInfo["pageCount"].(float64); ok && pagesRaw > 0 {
		pages := int(pagesRaw)
		book.PageCount = &pages
	}

	// Genres (categories)
	if catsRaw, ok := volumeInfo["categories"].([]interface{}); ok && len(catsRaw) > 0 {
		var cats []string
		for _, c := range catsRaw {
			if cat, ok := c.(string); ok && cat != "" {
				cats = append(cats, cat)
			}
		}
		if len(cats) > 0 {
			catsJSON, _ := json.Marshal(cats)
			s := string(catsJSON)
			book.Genres = &s
		}
	}

	// Cover image
	if imageLinksRaw, ok := volumeInfo["imageLinks"].(map[string]interface{}); ok {
		if thumbnail, ok := imageLinksRaw["thumbnail"].(string); ok && thumbnail != "" {
			book.CoverImageURL = &thumbnail
		}
	}

	return book, "google_books", nil
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

	// Resolve author keys in parallel
	if len(keysToResolve) > 0 {
		results := make(chan string, len(keysToResolve))
		var wg sync.WaitGroup
		for _, key := range keysToResolve {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				if name := fetchOpenLibraryAuthorName(k); name != "" {
					results <- name
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
	u := fmt.Sprintf("https://openlibrary.org%s.json", key)
	resp, err := apiHTTPClient.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}

	if name, ok := data["name"].(string); ok {
		return name
	}
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
		"title":              true,
		"subtitle":           true,
		"authors":            true,
		"illustrators":       true,
		"publisher":          true,
		"publication_year":   true,
		"page_count":         true,
		"book_type":          true,
		"reading_levels":     true,
		"genres":             true,
		"themes":             true,
		"awards":             true,
		"gift_from":          true,
		"gift_relationship":  true,
		"child_rating":       true,
		"read_count":         true,
		"cover_image_url":    true,
		"cover_source":       true,
		"isbn":               false,
		"condition":          false,
		"location":           false,
		"notes":              false,
		"date_received":      false,
		"last_read_date":     false,
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
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		isbn := strings.ReplaceAll(strings.TrimSpace(r.FormValue("isbn")), "-", "")
		if isbn == "" {
			http.Error(w, "ISBN is required", http.StatusBadRequest)
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
				child_rating, read_count, last_read_date, cover_image_url, cover_source, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)
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
			defaultGuestVisibleFields(),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				http.Error(w, "A book with this ISBN already exists", http.StatusConflict)
				return
			}
			http.Error(w, "Failed to create book", http.StatusInternalServerError)
			return
		}

		id, _ := result.LastInsertId()
		http.Redirect(w, r, "/books/"+strconv.FormatInt(id, 10), http.StatusFound)
	}
}

// HTMLUpdateBookHandler handles POST /books/{id}/update (form submission from the edit-book page).
func HTMLUpdateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid book ID", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}
		isbn := strings.ReplaceAll(strings.TrimSpace(r.FormValue("isbn")), "-", "")
		if isbn == "" {
			http.Error(w, "ISBN is required", http.StatusBadRequest)
			return
		}

		// Build dynamic UPDATE from form fields
		sets := []string{}
		args := []interface{}{}

		formFields := []string{
			"isbn",            // handled specially above
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

		// Always update timestamp
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, id)

		// #nosec G202 -- Column names are hardcoded
		query := "UPDATE books SET " + strings.Join(sets, ", ") + " WHERE id = ?"
		result, err := db.Exec(query, args...)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				http.Error(w, "A book with this ISBN already exists", http.StatusConflict)
				return
			}
			http.Error(w, "Failed to update book", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/books/"+strconv.FormatInt(id, 10), http.StatusFound)
	}
}

// ptrIfNonEmpty returns a pointer to s if s is non-empty, nil otherwise.
func ptrIfNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
