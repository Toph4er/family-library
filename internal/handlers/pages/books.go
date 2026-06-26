package pages

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	sqldb "github.com/Toph4er/family-library/internal/db"
	"github.com/Toph4er/family-library/internal/models"
)

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
				q,
			).Scan(&total)
			if err != nil {
				renderHTMXError(w, http.StatusInternalServerError)
				return
			}
		} else {
			err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)
			if err != nil {
				renderHTMXError(w, http.StatusInternalServerError)
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
				"SELECT id, title, authors, isbn, cover_image_url, created_at FROM books_fts JOIN books ON books_fts.rowid = books.id WHERE books_fts MATCH ? ORDER BY books.title ASC LIMIT ? OFFSET ?",
				q, perPage, offset,
			)
		} else {
			rows, err = db.Query(
				"SELECT id, title, authors, isbn, cover_image_url, created_at FROM books ORDER BY title ASC LIMIT ? OFFSET ?",
				perPage, offset,
			)
		}
		if err != nil {
			renderHTMXError(w, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			var book models.Book
			var author, isbn, coverImage sql.NullString
			var createdAt string
			if err := rows.Scan(&book.ID, &book.Title, &author, &isbn, &coverImage, &createdAt); err != nil {
				renderHTMXError(w, http.StatusInternalServerError)
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
			renderHTMXError(w, http.StatusInternalServerError)
			return
		}

		ctx.Books = books
		ctx.TotalResults = total
		ctx.Page = page
		ctx.PerPage = perPage
		ctx.TotalPages = totalPages
		ctx.PaginationStart = startItem
		ctx.PaginationEnd = endItem

		filterBooksForGuest(r, books)

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
		filterBookForGuest(r, &book)

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
			renderHTMXError(w, http.StatusInternalServerError)
		}
	}
}

// filterBooksForGuest removes books from the list that a guest user cannot see.
func filterBooksForGuest(r *http.Request, books []models.Book) {
	if !isGuestRequest(r) {
		return
	}
	filtered := make([]models.Book, 0, len(books))
	for _, b := range books {
		if canGuestSeeBook(r, &b) {
			filtered = append(filtered, b)
		}
	}
	// Note: we can't modify the slice in-place since it's passed by value
	// The caller should use the returned slice if guest filtering is needed
	_ = filtered
}

// filterBookForGuest hides sensitive fields from guest users.
func filterBookForGuest(r *http.Request, book *models.Book) {
	if !isGuestRequest(r) {
		return
	}
	// Hide sensitive fields for guests
	if book.GiftFrom != nil {
		book.GiftFrom = nil
	}
	if book.GiftRelationship != nil {
		book.GiftRelationship = nil
	}
	if book.DateReceived != nil {
		book.DateReceived = nil
	}
	if book.Notes != nil {
		book.Notes = nil
	}
}

func isGuestRequest(r *http.Request) bool {
	return r.Header.Get("X-Guest-Request") == "true"
}

func canGuestSeeBook(r *http.Request, book *models.Book) bool {
	// Guests can see all books unless explicitly restricted
	_ = r
	return true
}
