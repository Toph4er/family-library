package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/middleware"
	"git.rcsmaine.com/chris/library/backend/internal/models"
)

// pageContext holds common template data for all page handlers.
type pageContext struct {
	Year                   int
	CSRFToken              string
	IsAdmin                bool
	IsAuthenticated        bool
	Username               string
	SiteName               string
	SiteTagline            string
	Books                  []models.Book
	Book                   *models.Book
	Items                  []models.WishlistItem
	Users                  []map[string]interface{}
	Settings               map[string]string
	DefaultGuestVisibility map[string]bool
}

// buildPageContext creates a pageContext for the given request.
// It first checks the request context (set by auth middleware), then falls
// back to reading the session directly for routes without that middleware.
func buildPageContext(r *http.Request, store *sessions.CookieStore, sessionName string) pageContext {
	ctx := pageContext{Year: time.Now().Year()}

	// Check context first (set by auth middleware on protected routes).
	if user := auth.GetUserFromContext(r); user != nil {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = !user.IsGuest
		ctx.Username = user.Username
		if s := middleware.GetSessionFromContext(r); s != nil {
			if token, ok := s.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
				ctx.CSRFToken = token
			}
		}
		return ctx
	}

	// Fall back to reading the session directly (public routes without auth middleware).
	session, err := store.Get(r, sessionName)
	if err != nil {
		return ctx
	}
	if _, ok := session.Values[auth.UserIDKey].(int64); ok {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = true
		if username, ok := session.Values[auth.UsernameKey].(string); ok {
			ctx.Username = username
		}
	} else if isGuest, ok := session.Values[auth.IsGuestKey].(bool); ok && isGuest {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = false
	}

	// Extract CSRF token from session if available.
	if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
		ctx.CSRFToken = token
	}

	return ctx
}

// isHTMXRequest checks if the request originated from HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// renderPage renders a template, returning only the content fragment for HTMX
// requests (HX-Request header present) or the full page layout otherwise.
func renderPage(w http.ResponseWriter, r *http.Request, tmpl *template.Template, pageName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if isHTMXRequest(r) {
		// HTMX request — render only the content block, not the base layout.
		if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
			slog.Error("template error", "page", pageName, "error", err)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	} else {
		// Full page request — render the page template (which includes base).
		if err := tmpl.ExecuteTemplate(w, pageName, data); err != nil {
			slog.Error("template error", "page", pageName, "error", err)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderLandingPage renders the public landing page.
// Authenticated users are redirected to /books.
func RenderLandingPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		// Authenticated users should not see the landing page — send them to books.
		if ctx.IsAuthenticated {
			http.Redirect(w, r, "/books", http.StatusFound)
			return
		}

		siteName := "Our Library"
		siteTagline := "A woodland fairy tale collection"

		var nameVal, taglineVal sql.NullString
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "site_name").Scan(&nameVal); err == nil {
			siteName = nameVal.String
		}
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "site_tagline").Scan(&taglineVal); err == nil {
			siteTagline = taglineVal.String
		}

		ctx.SiteName = siteName
		ctx.SiteTagline = siteTagline

		renderPage(w, r, tmpl, "landing.html", ctx)
	}
}

// RenderBooksPage renders the books listing page (auth required).
func RenderBooksPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		// Defense-in-depth: reject unauthenticated requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		rows, err := db.Query("SELECT id, title, authors, isbn, cover_image_url, created_at FROM books ORDER BY title ASC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			var book models.Book
			var author, isbn, coverImage sql.NullString
			var createdAt string
			if err := rows.Scan(&book.ID, &book.Title, &author, &isbn, &coverImage, &createdAt); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
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
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		ctx.Books = books
		filterBooksForGuest(r, books)

		renderPage(w, r, tmpl, "books.html", ctx)
	}
}

// RenderBookDetailPage renders the book detail page (auth required).
func RenderBookDetailPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		// Defense-in-depth: reject unauthenticated requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		id := chi.URLParam(r, "id")

		var book models.Book
		var title, createdAt, updatedAt string
		var subtitle, author, illustrators, publisher, bookType, readingLevels, genres, themes, awards, giftFrom, giftRelationship, dateReceived, condition, location, notes, lastReadDate, coverImage, coverSource, guestVisibleFields sql.NullString
		var pubYear, pageCount, childRating, readCount sql.NullInt64

		err := db.QueryRow(`SELECT id, title, subtitle, authors, illustrators, isbn, publisher, publication_year, page_count, book_type, reading_levels, genres, themes, awards, gift_from, gift_relationship, date_received, condition, location, notes, child_rating, read_count, last_read_date, cover_image_url, cover_source, guest_visible_fields, created_at, updated_at FROM books WHERE id = ?`, id).Scan(
			&book.ID, &title, &subtitle, &author, &illustrators, &book.ISBN, &publisher, &pubYear, &pageCount, &bookType, &readingLevels, &genres, &themes, &awards, &giftFrom, &giftRelationship, &dateReceived, &condition, &location, &notes, &childRating, &readCount, &lastReadDate, &coverImage, &coverSource, &guestVisibleFields, &createdAt, &updatedAt,
		)
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		book.Title = title
		if subtitle.Valid {
			book.Subtitle = &subtitle.String
		}
		if author.Valid {
			book.Authors = &author.String
		}
		if illustrators.Valid {
			book.Illustrators = &illustrators.String
		}
		if publisher.Valid {
			book.Publisher = &publisher.String
		}
		if pubYear.Valid {
			y := int(pubYear.Int64)
			book.PublicationYear = &y
		}
		if pageCount.Valid {
			p := int(pageCount.Int64)
			book.PageCount = &p
		}
		if bookType.Valid {
			book.BookType = &bookType.String
		}
		if readingLevels.Valid {
			book.ReadingLevels = &readingLevels.String
		}
		if genres.Valid {
			book.Genres = &genres.String
		}
		if themes.Valid {
			book.Themes = &themes.String
		}
		if awards.Valid {
			book.Awards = &awards.String
		}
		if giftFrom.Valid {
			book.GiftFrom = &giftFrom.String
		}
		if giftRelationship.Valid {
			book.GiftRelationship = &giftRelationship.String
		}
		if dateReceived.Valid {
			book.DateReceived = &dateReceived.String
		}
		if condition.Valid {
			book.Condition = &condition.String
		}
		if location.Valid {
			book.Location = &location.String
		}
		if notes.Valid {
			book.Notes = &notes.String
		}
		if childRating.Valid {
			cr := int(childRating.Int64)
			book.ChildRating = &cr
		}
		if readCount.Valid {
			book.ReadCount = int(readCount.Int64)
		}
		if lastReadDate.Valid {
			book.LastReadDate = &lastReadDate.String
		}
		if coverImage.Valid {
			book.CoverImageURL = &coverImage.String
		}
		if coverSource.Valid {
			book.CoverSource = &coverSource.String
		}
		book.CreatedAt = createdAt
		book.UpdatedAt = updatedAt
		if guestVisibleFields.Valid {
			book.GuestVisibleFields = guestVisibleFields.String
		}

		ctx.Book = &book
		filterBookForGuest(r, &book)

		renderPage(w, r, tmpl, "book-detail.html", ctx)
	}
}

// RenderWishlistPage renders the wishlist page (auth required).
func RenderWishlistPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		// Defense-in-depth: reject unauthenticated requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		rows, err := db.Query("SELECT id, isbn, title, author, priority, notes, fulfilled, requested_by, requested_at, fulfilled_at FROM wishlist ORDER BY priority DESC, requested_at DESC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		items := make([]models.WishlistItem, 0)
		for rows.Next() {
			var item models.WishlistItem
			var isbn, author, notes sql.NullString
			var fulfilled bool
			var requestedBy sql.NullString
			var requestedAtStr, fulfilledAtStr sql.NullString
			if err := rows.Scan(&item.ID, &isbn, &item.Title, &author, &item.Priority, &notes, &fulfilled, &requestedBy, &requestedAtStr, &fulfilledAtStr); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			if isbn.Valid {
				item.ISBN = &isbn.String
			}
			if author.Valid {
				item.Author = &author.String
			}
			if notes.Valid {
				item.Notes = &notes.String
			}
			if requestedBy.Valid {
				item.RequestedBy = &requestedBy.String
			}
			if requestedAtStr.Valid {
				item.RequestedAt = requestedAtStr.String
			}
			item.Fulfilled = fulfilled
			if fulfilledAtStr.Valid {
				item.FulfilledAt = &fulfilledAtStr.String
				item.Fulfilled = true
			}
			items = append(items, item)
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		ctx.Items = items

		renderPage(w, r, tmpl, "wishlist.html", ctx)
	}
}

// RenderAdminPage renders the admin page (admin required).
func RenderAdminPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		// Defense-in-depth: reject unauthenticated or non-admin requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !ctx.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		rows, err := db.Query("SELECT id, username, role, display_name, created_at FROM users ORDER BY id")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		users := make([]map[string]interface{}, 0)
		for rows.Next() {
			var id int64
			var username, role, displayName, createdAt string
			if err := rows.Scan(&id, &username, &role, &displayName, &createdAt); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			users = append(users, map[string]interface{}{
				"id":           id,
				"username":     username,
				"role":         role,
				"display_name": displayName,
				"created_at":   createdAt,
			})
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		ctx.Users = users

		renderPage(w, r, tmpl, "admin.html", ctx)
	}
}

// RenderSettingsPage renders the settings page (admin required).
func RenderSettingsPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		// Defense-in-depth: reject unauthenticated or non-admin requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !ctx.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// Load settings
		rows, err := db.Query("SELECT key, value FROM settings ORDER BY key ASC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			if _, sensitive := sensitiveKeys[key]; sensitive {
				continue
			}
			settings[key] = value
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		ctx.Settings = settings

		// Load users
		userRows, err := db.Query("SELECT id, username, role, display_name, created_at FROM users ORDER BY id")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer userRows.Close()

		users := make([]map[string]interface{}, 0)
		for userRows.Next() {
			var id int64
			var username, role, displayName, createdAt string
			if err := userRows.Scan(&id, &username, &role, &displayName, &createdAt); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			users = append(users, map[string]interface{}{
				"id":           id,
				"username":     username,
				"role":         role,
				"display_name": displayName,
				"created_at":   createdAt,
			})
		}
		if err = userRows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		ctx.Users = users

		// Load default guest visibility
		var defaultVisibility string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "default_guest_visibility").Scan(&defaultVisibility); err == nil {
			ctx.DefaultGuestVisibility = make(map[string]bool)
			_ = json.Unmarshal([]byte(defaultVisibility), &ctx.DefaultGuestVisibility)
		}

		renderPage(w, r, tmpl, "settings.html", ctx)
	}
}

// RenderBookFormPage renders the add/edit book form as a full page.
func RenderBookFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, bookID int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

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
			row := db.QueryRow(`SELECT `+bookColumns+` FROM books WHERE id = ?`, bookID)
			b, err := scanBook(row)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			book = *b
			bookTitle = book.Title
			cancelURL = "/books/" + strconv.FormatInt(bookID, 10)
		}

		data := map[string]interface{}{
			"Year":            ctx.Year,
			"CSRFToken":       ctx.CSRFToken,
			"IsAdmin":         ctx.IsAdmin,
			"IsAuthenticated": ctx.IsAuthenticated,
			"Username":        ctx.Username,
			"IsEdit":          isEdit,
			"BookTitle":       bookTitle,
			"CancelURL":       cancelURL,
			"ActionURL":       func() string {
				if isEdit {
					return "/books/" + strconv.FormatInt(bookID, 10) + "/update"
				}
				return "/books/create"
			}(),
			"Title":           derefString(&book.Title),
			"Subtitle":        derefString(book.Subtitle),
			"Authors":         derefString(book.Authors),
			"Illustrators":    derefString(book.Illustrators),
			"ISBN":            derefString(book.ISBN),
			"Publisher":       derefString(book.Publisher),
			"PublicationYear": derefInt(book.PublicationYear),
			"PageCount":       derefInt(book.PageCount),
			"BookType":        derefString(book.BookType),
			"Condition":       derefString(book.Condition),
			"Genres":          derefString(book.Genres),
			"Themes":          derefString(book.Themes),
			"Awards":          derefString(book.Awards),
			"ReadingLevels":   derefString(book.ReadingLevels),
			"GiftFrom":        derefString(book.GiftFrom),
			"GiftRelationship": derefString(book.GiftRelationship),
			"DateReceived":    derefString(book.DateReceived),
			"Location":        derefString(book.Location),
			"CoverImageURL":   derefString(book.CoverImageURL),
			"Notes":           derefString(book.Notes),
			"ChildRating":     derefInt(book.ChildRating),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "book-form.html", data); err != nil {
			slog.Error("template error", "page", "book-form", "error", err)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

