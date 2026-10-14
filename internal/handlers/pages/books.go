package pages

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	sqldb "github.com/Toph4er/family-library/internal/db"
	"github.com/Toph4er/family-library/internal/models"
)

// SafeFTS5Query converts a raw user search string into a syntactically
// safe FTS5 MATCH expression. Each whitespace-separated token is wrapped in
// double quotes (embedded double quotes are doubled to escape) and suffixed
// with `*`, so the input is treated as literal prefix phrases joined by
// implicit AND (e.g. `bear` matches `bear`, `bears`). Without the quoting,
// characters like apostrophes, ':', '-', '*', or parentheses are parsed as
// FTS5 syntax and produce errors such as: fts5: syntax error near "'".
func SafeFTS5Query(q string) string {
	fields := strings.Fields(q)
	for i, f := range fields {
		fields[i] = `"` + strings.ReplaceAll(f, `"`, `""`) + `"*`
	}
	return strings.Join(fields, " ")
}

// RenderBooksPage renders the books listing page (auth required).
// Supports ?q= search parameter for filtering books.
func RenderBooksPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q != "" {
			ctx.CurrentQuery = q
		}

		// Pagination params
		page := 1
		perPage := 20
		if p := r.URL.Query().Get("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		if p := r.URL.Query().Get("per_page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 && v <= 100 {
				perPage = v
			}
		}
		offset := (page - 1) * perPage

		// Count query — use FTS5 when searching.
		var total int
		if q != "" {
			err := db.QueryRow(
				"SELECT COUNT(*) FROM books_fts JOIN books ON books_fts.rowid = books.id WHERE books_fts MATCH ?",
				SafeFTS5Query(q),
			).Scan(&total)
			if err != nil {
				renderHTMXError(w, r, http.StatusInternalServerError)
				return
			}
		} else {
			err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)
			if err != nil {
				renderHTMXError(w, r, http.StatusInternalServerError)
				return
			}
		}

		totalPages := (total + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}
		// Clamp page to valid range
		if page > totalPages {
			page = totalPages
			offset = (page - 1) * perPage
		}

		startItem := offset + 1
		endItem := offset + perPage
		if endItem > total {
			endItem = total
		}
		if total == 0 {
			startItem = 0
			endItem = 0
		}

		// Data query with LIMIT/OFFSET — use FTS5 when searching.
		var rows *sql.Rows
		var err error
		if q != "" {
			rows, err = db.Query(
				"SELECT books.id, books.title, books.authors, books.isbn, books.cover_image_url, books.created_at FROM books_fts JOIN books ON books_fts.rowid = books.id WHERE books_fts MATCH ? ORDER BY books.title ASC LIMIT ? OFFSET ?",
				SafeFTS5Query(q), perPage, offset,
			)
		} else {
			rows, err = db.Query(
				"SELECT id, title, authors, isbn, cover_image_url, created_at FROM books ORDER BY title ASC LIMIT ? OFFSET ?",
				perPage, offset,
			)
		}
		if err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			var book models.Book
			var author, isbn, coverImage sql.NullString
			var createdAt string
			if err := rows.Scan(&book.ID, &book.Title, &author, &isbn, &coverImage, &createdAt); err != nil {
				renderHTMXError(w, r, http.StatusInternalServerError)
				return
			}
			if author.Valid {
				book.Authors = &author.String
			}
			if isbn.Valid {
				book.ISBN = &isbn.String
			}
			if coverImage.Valid {
				book.CoverImageURL = &coverImage.String
			}
			book.CreatedAt = createdAt
			books = append(books, book)
		}
		if err = rows.Err(); err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}

		ctx.Books = books
		ctx.TotalResults = total
		ctx.Page = page
		ctx.PerPage = perPage
		ctx.TotalPages = totalPages
		ctx.PaginationStart = startItem
		ctx.PaginationEnd = endItem

		// Apply the admin's guest visibility to guests only: pointer fields
		// the admin marked non-visible are nulled so the templates' {{if}}
		// guards hide them. Admins see the full record (ctx.IsGuest is false).
		if ctx.IsGuest {
			if vis := loadGuestVisibility(db); vis != nil {
				for i := range books {
					books[i].ApplyVisibility(vis)
				}
			}
		}

		renderPage(w, r, tmpl, "books.html", ctx)
	}
}

// RenderBookDetailPage renders the book detail page (auth required).
func RenderBookDetailPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		// Defense-in-depth: reject unauthenticated requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		id := chi.URLParam(r, "id")

		var book models.Book
		row := db.QueryRow(`SELECT `+sqldb.BookColumns+` FROM books WHERE id = ?`, id) // #nosec G202 -- BookColumns is a constant
		b, err := sqldb.ScanBook(row)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		book = *b

		ctx.Book = &book
		// Apply the admin's guest visibility to guests only — pointer fields
		// the admin marked non-visible are nulled so the template's {{if}}
		// guards hide them. Admins see the full record (ctx.IsGuest is false).
		if ctx.IsGuest {
			if vis := loadGuestVisibility(db); vis != nil {
				book.ApplyVisibility(vis)
			}
		}

		renderPage(w, r, tmpl, "book-detail.html", ctx)
	}
}

// RenderBookFormPage renders the add/edit book form as a full page.
func RenderBookFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, bookID int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !ctx.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		var book models.Book
		bookTitle := ""
		cancelURL := "/books"

		if isEdit {
			row := db.QueryRow(`SELECT `+sqldb.BookColumns+` FROM books WHERE id = ?`, bookID) // #nosec G202 -- BookColumns is a constant
			b, err := sqldb.ScanBook(row)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			book = *b
			bookTitle = book.Title
			cancelURL = "/books/" + strconv.FormatInt(bookID, 10)
		}

		data := pageContext{
			BaseContext: ctx.BaseContext,
			BookFormContext: BookFormContext{
				BookID:    bookID,
				IsEdit:    isEdit,
				BookTitle: bookTitle,
				CancelURL: cancelURL,
				ActionURL: func() string {
					if isEdit {
						return "/books/" + strconv.FormatInt(bookID, 10) + "/update"
					}
					return "/books/create"
				}(),
				Title:        derefString(&book.Title),
				Subtitle:     derefString(book.Subtitle),
				Authors:      derefString(book.Authors),
				Illustrators: derefString(book.Illustrators),
				ISBN:         derefString(book.ISBN),
				Publisher:    derefString(book.Publisher),
				PublicationYear: func() string {
					if book.PublicationYear != nil {
						return strconv.Itoa(*book.PublicationYear)
					}
					return ""
				}(),
				PageCount: func() string {
					if book.PageCount != nil {
						return strconv.Itoa(*book.PageCount)
					}
					return ""
				}(),
				BookType:          derefString(book.BookType),
				Condition:         derefString(book.Condition),
				Genres:            derefString(book.Genres),
				Themes:            derefString(book.Themes),
				Awards:            derefString(book.Awards),
				ReadingLevels:     derefString(book.ReadingLevels),
				DeweyDecimalClass: derefString(book.DeweyDecimalClass),
				Language:          derefString(book.Language),
				Series:            derefString(book.Series),
				AgeRange:          derefString(book.AgeRange),
				SubjectPlaces:     derefString(book.SubjectPlaces),
				SubjectPeople:     derefString(book.SubjectPeople),
				SubjectTimes:      derefString(book.SubjectTimes),
				Description:       derefString(book.Description),
				GiftFrom:          derefString(book.GiftFrom),
				GiftRelationship:  derefString(book.GiftRelationship),
				DateReceived:      derefString(book.DateReceived),
				Location:          derefString(book.Location),
				CoverImageURL:     derefString(book.CoverImageURL),
				Notes:             derefString(book.Notes),
				ChildRating:       derefInt(book.ChildRating),
				Quantity: func() int {
					if book.Quantity > 0 {
						return book.Quantity
					}
					return 1
				}(),
			},
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "book-form.html", data); err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
		}
	}
}

// loadGuestVisibility reads the admin's default_guest_visibility setting — the
// same blob the settings UI edits under "Guest book visibility". It maps each
// book field to whether it is visible to guests. Returns nil when the setting
// is absent or malformed, in which case callers leave the book unfiltered
// (matching FilterForGuest's empty-blob = show-all convention).
func loadGuestVisibility(db *sql.DB) map[string]bool {
	if db == nil {
		return nil
	}
	var value string
	if err := db.QueryRow(
		"SELECT value FROM settings WHERE key = 'default_guest_visibility'",
	).Scan(&value); err != nil {
		return nil
	}
	if value == "" {
		return nil
	}
	var vis map[string]bool
	if err := json.Unmarshal([]byte(value), &vis); err != nil {
		return nil
	}
	return vis
}

// BookSearchHandler returns a JSON list of books matching the FTS5 search query.
// GET /books/search?q=...
func BookSearchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]models.Book{}); err != nil {
				log.Printf("error encoding empty book list: %v", err)
				return
			}
		}

		rows, err := db.Query(
			"SELECT books.id, books.title, books.authors, books.source FROM books_fts JOIN books ON books_fts.rowid = books.id WHERE books_fts MATCH ? ORDER BY books.title ASC LIMIT 20",
			SafeFTS5Query(q),
		)
		if err != nil {
			log.Printf("book search query failed: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var books []models.Book
		for rows.Next() {
			var b models.Book
			var authors sql.NullString
			if err := rows.Scan(&b.ID, &b.Title, &authors, &b.Source); err != nil {
				continue
			}
			if authors.Valid {
				b.Authors = &authors.String
			}
			books = append(books, b)
		}
		if books == nil {
			books = []models.Book{}
		}

		// This endpoint is public, so whatever it returns is guest-visible:
		// apply the admin's guest visibility (nil out authors when the admin
		// has marked authors non-visible).
		if vis := loadGuestVisibility(db); vis != nil && !vis["authors"] {
			for i := range books {
				books[i].Authors = nil
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(books); err != nil {
			log.Printf("error encoding books: %v", err)
		}
	}
}
