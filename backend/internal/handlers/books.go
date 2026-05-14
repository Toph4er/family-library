package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	b.id, b.isbn, b.title, b.subtitle, b.authors, b.illustrators,
	b.publisher, b.publication_year, b.page_count, b.book_type,
	b.reading_levels, b.genres, b.themes, b.awards,
	b.gift_from, b.gift_relationship, b.date_received,
	b.condition, b.location, b.notes,
	b.child_rating, b.read_count, b.last_read_date,
	b.cover_image_url, b.cover_source, b.guest_visible_fields,
	b.created_at, b.updated_at
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
		if err == sql.ErrNoRows {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

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
			req.ISBN, req.Title, req.Subtitle, req.Authors, req.Illustrators,
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

		// Build dynamic UPDATE from non-nil fields
		sets := []string{}
		args := []interface{}{}

		if req.ISBN != nil {
			sets = append(sets, "isbn = ?")
			args = append(args, *req.ISBN)
		}
		if req.Title != nil {
			sets = append(sets, "title = ?")
			args = append(args, *req.Title)
		}
		if req.Subtitle != nil {
			sets = append(sets, "subtitle = ?")
			args = append(args, *req.Subtitle)
		}
		if req.Authors != nil {
			sets = append(sets, "authors = ?")
			args = append(args, *req.Authors)
		}
		if req.Illustrators != nil {
			sets = append(sets, "illustrators = ?")
			args = append(args, *req.Illustrators)
		}
		if req.Publisher != nil {
			sets = append(sets, "publisher = ?")
			args = append(args, *req.Publisher)
		}
		if req.PublicationYear != nil {
			sets = append(sets, "publication_year = ?")
			args = append(args, *req.PublicationYear)
		}
		if req.PageCount != nil {
			sets = append(sets, "page_count = ?")
			args = append(args, *req.PageCount)
		}
		if req.BookType != nil {
			sets = append(sets, "book_type = ?")
			args = append(args, *req.BookType)
		}
		if req.ReadingLevels != nil {
			sets = append(sets, "reading_levels = ?")
			args = append(args, *req.ReadingLevels)
		}
		if req.Genres != nil {
			sets = append(sets, "genres = ?")
			args = append(args, *req.Genres)
		}
		if req.Themes != nil {
			sets = append(sets, "themes = ?")
			args = append(args, *req.Themes)
		}
		if req.Awards != nil {
			sets = append(sets, "awards = ?")
			args = append(args, *req.Awards)
		}
		if req.GiftFrom != nil {
			sets = append(sets, "gift_from = ?")
			args = append(args, *req.GiftFrom)
		}
		if req.GiftRelationship != nil {
			sets = append(sets, "gift_relationship = ?")
			args = append(args, *req.GiftRelationship)
		}
		if req.DateReceived != nil {
			sets = append(sets, "date_received = ?")
			args = append(args, *req.DateReceived)
		}
		if req.Condition != nil {
			sets = append(sets, "condition = ?")
			args = append(args, *req.Condition)
		}
		if req.Location != nil {
			sets = append(sets, "location = ?")
			args = append(args, *req.Location)
		}
		if req.Notes != nil {
			sets = append(sets, "notes = ?")
			args = append(args, *req.Notes)
		}
		if req.ChildRating != nil {
			sets = append(sets, "child_rating = ?")
			args = append(args, *req.ChildRating)
		}
		if req.ReadCount != nil {
			sets = append(sets, "read_count = ?")
			args = append(args, *req.ReadCount)
		}
		if req.LastReadDate != nil {
			sets = append(sets, "last_read_date = ?")
			args = append(args, *req.LastReadDate)
		}
		if req.CoverImageURL != nil {
			sets = append(sets, "cover_image_url = ?")
			args = append(args, *req.CoverImageURL)
		}
		if req.CoverSource != nil {
			sets = append(sets, "cover_source = ?")
			args = append(args, *req.CoverSource)
		}

		// Always update the timestamp
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")

		if len(sets) == 1 {
			// Only updated_at was set — nothing to do
			row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, id)
			b, err := scanBook(row)
			if err == sql.ErrNoRows {
				JSONError(w, http.StatusNotFound, "book not found")
				return
			}
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			JSONResponse(w, http.StatusOK, b)
			return
		}

		args = append(args, id)
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
		isbn := strings.TrimSpace(req.ISBN)
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

		// 3. Try Google Books API
		book, coverSource, apiErr := fetchFromGoogleBooks(isbn)
		if apiErr != nil || book == nil {
			// 4. Fallback to Open Library API
			book, coverSource, apiErr = fetchFromOpenLibrary(isbn)
			if apiErr != nil {
				JSONError(w, http.StatusBadGateway, "both book APIs failed")
				return
			}
			if book == nil {
				JSONError(w, http.StatusNotFound, "book not found by either API")
				return
			}
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

// fetchFromGoogleBooks fetches book metadata from the Google Books API.
// Returns the populated book, cover source string, and any error.
// If no results are found, returns (nil, "", nil).
func fetchFromGoogleBooks(isbn string) (*models.Book, string, error) {
	u := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=isbn:%s", url.QueryEscape(isbn))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("building Google Books request: %w", err)
	}
	req.Header.Set("User-Agent", "Library-Book-Collection/1.0")

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("Google Books HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading Google Books response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Non-200 from Google — treat as "not found" so we can fallback
		return nil, "", nil
	}

	var googleResp struct {
		Items []struct {
			VolumeInfo struct {
				Title       string   `json:"title"`
				Subtitle    string   `json:"subtitle"`
				Authors     []string `json:"authors"`
				Publisher   string   `json:"publisher"`
				PublishedDate string `json:"publishedDate"`
				PageCount   int      `json:"pageCount"`
				Categories  []string `json:"categories"`
				ImageLinks  struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"imageLinks"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &googleResp); err != nil {
		return nil, "", fmt.Errorf("parsing Google Books JSON: %w", err)
	}

	if len(googleResp.Items) == 0 {
		return nil, "", nil
	}

	vi := googleResp.Items[0].VolumeInfo
	book := &models.Book{
		Title: vi.Title,
	}
	if vi.Subtitle != "" {
		book.Subtitle = &vi.Subtitle
	}
	if len(vi.Authors) > 0 {
		authorsJSON, _ := json.Marshal(vi.Authors)
		s := string(authorsJSON)
		book.Authors = &s
	}
	if vi.Publisher != "" {
		book.Publisher = &vi.Publisher
	}
	if m := yearRe.FindStringSubmatch(vi.PublishedDate); m != nil {
		year, _ := strconv.Atoi(m[1])
		book.PublicationYear = &year
	}
	if vi.PageCount > 0 {
		book.PageCount = &vi.PageCount
	}
	if len(vi.Categories) > 0 {
		genresJSON, _ := json.Marshal(vi.Categories)
		s := string(genresJSON)
		book.Genres = &s
	}
	if vi.ImageLinks.Thumbnail != "" {
		// Google Books placeholder URL — skip it
		if !strings.Contains(vi.ImageLinks.Thumbnail, "placeholder") {
			book.CoverImageURL = &vi.ImageLinks.Thumbnail
		}
	}

	return book, "google_books", nil
}

// fetchFromOpenLibrary fetches book metadata from the Open Library API.
// Returns the populated book, cover source string, and any error.
// If no results are found, returns (nil, "", nil).
func fetchFromOpenLibrary(isbn string) (*models.Book, string, error) {
	u := fmt.Sprintf("https://openlibrary.org/isbn/%s.json", url.PathEscape(isbn))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", fmt.Errorf("building Open Library request: %w", err)
	}

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

	// Authors: array of {name: "..."}
	if authorsRaw, ok := olResp["authors"].([]interface{}); ok && len(authorsRaw) > 0 {
		var authors []string
		for _, a := range authorsRaw {
			if m, ok := a.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					authors = append(authors, name)
				}
			}
		}
		if len(authors) > 0 {
			authorsJSON, _ := json.Marshal(authors)
			s := string(authorsJSON)
			book.Authors = &s
		}
	}

	// Publisher: array of strings
	if publishersRaw, ok := olResp["publisher"].([]interface{}); ok && len(publishersRaw) > 0 {
		if pub, ok := publishersRaw[0].(string); ok && pub != "" {
			book.Publisher = &pub
		}
	}

	// Publish date: extract year
	if publishDate, ok := olResp["publish_date"].(string); ok {
		if m := yearRe.FindStringSubmatch(publishDate); m != nil {
			year, _ := strconv.Atoi(m[1])
			book.PublicationYear = &year
		}
	}

	// Page count
	if pagesRaw, ok := olResp["number_of_pages_num"].(float64); ok && pagesRaw > 0 {
		pages := int(pagesRaw)
		book.PageCount = &pages
	}

	// Cover image
	if coverIDRaw, ok := olResp["cover_i"].(float64); ok && coverIDRaw > 0 {
		coverURL := fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", int(coverIDRaw))
		book.CoverImageURL = &coverURL
	}

	return book, "open_library", nil
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
