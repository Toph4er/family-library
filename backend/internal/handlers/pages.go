package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/middleware"
	"git.rcsmaine.com/chris/library/backend/internal/models"
)

// pageContext holds common template data for all page handlers.
type pageContext struct {
	Year            int
	CSRFToken       string
	IsAdmin         bool
	IsAuthenticated bool
	SiteName        string
	SiteTagline     string
	Books           []models.Book
	Book            *models.Book
	Items           []models.WishlistItem
	Users           []map[string]interface{}
	Settings        map[string]string
}

// buildPageContext creates a pageContext for the given request.
func buildPageContext(r *http.Request, store *sessions.CookieStore, sessionName string) pageContext {
	user := auth.GetUserFromContext(r)
	ctx := pageContext{Year: time.Now().Year()}
	if user != nil {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = !user.IsGuest
		if s := middleware.GetSessionFromContext(r); s != nil {
			if token, ok := s.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
				ctx.CSRFToken = token
			}
		}
	}
	return ctx
}

// RenderLandingPage renders the public landing page.
func RenderLandingPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "landing.html", ctx); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderBooksPage renders the public books listing page.
func RenderBooksPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		rows, err := db.Query("SELECT id, title, authors, isbn, cover_image_url, created_at FROM books ORDER BY title ASC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			var book models.Book
			var author, isbn, coverImage, createdAt string
			if err := rows.Scan(&book.ID, &book.Title, &author, &isbn, &coverImage, &createdAt); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			book.Authors, _ = stringPtr(author)
			book.ISBN, _ = stringPtr(isbn)
			book.CoverImageURL, _ = stringPtr(coverImage)
			book.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
			books = append(books, book)
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		ctx.Books = books

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "books.html", ctx); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderBookDetailPage renders the public book detail page.
func RenderBookDetailPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		id := chi.URLParam(r, "id")

		var book models.Book
		var title, author, isbn, coverImage, notes, createdAt string
		err := db.QueryRow("SELECT id, title, authors, isbn, cover_image_url, notes, created_at FROM books WHERE id = ?", id).Scan(
			&book.ID, &title, &author, &isbn, &coverImage, &notes, &createdAt,
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
		book.Authors, _ = stringPtr(author)
		book.ISBN, _ = stringPtr(isbn)
		book.CoverImageURL, _ = stringPtr(coverImage)
		book.Notes, _ = stringPtr(notes)
		book.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

		ctx.Book = &book

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "book-detail.html", ctx); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderWishlistPage renders the wishlist page (auth required).
func RenderWishlistPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

		rows, err := db.Query("SELECT id, isbn, title, author, priority, notes, fulfilled, requested_by, requested_at, fulfilled_at FROM wishlist ORDER BY priority DESC, requested_at DESC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		items := make([]models.WishlistItem, 0)
		for rows.Next() {
			var item models.WishlistItem
			var isbn, author, notes string
			var fulfilled bool
			var requestedBy string
			var requestedAtStr, fulfilledAtStr string
			if err := rows.Scan(&item.ID, &isbn, &item.Title, &author, &item.Priority, &notes, &fulfilled, &requestedBy, &requestedAtStr, &fulfilledAtStr); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			item.ISBN, _ = stringPtr(isbn)
			item.Author, _ = stringPtr(author)
			item.Notes, _ = stringPtr(notes)
			item.Fulfilled = fulfilled
			item.RequestedBy, _ = stringPtr(requestedBy)
			item.RequestedAt, _ = time.Parse("2006-01-02 15:04:05", requestedAtStr)
			if fulfilledAtStr != "" {
				if t, err := time.Parse("2006-01-02 15:04:05", fulfilledAtStr); err == nil {
					item.FulfilledAt = &t
					item.Fulfilled = true
				}
			}
			items = append(items, item)
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		ctx.Items = items

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "wishlist.html", ctx); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderAdminPage renders the admin page (admin required).
func RenderAdminPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "admin.html", ctx); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderSettingsPage renders the settings page (admin required).
func RenderSettingsPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName)

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
			switch key {
			case "admin_password", "guest_password", "jwt_secret":
				continue
			}
			settings[key] = value
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		ctx.Settings = settings

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "settings.html", ctx); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

func stringPtr(s string) (*string, error) {
	if s == "" {
		return nil, nil
	}
	return &s, nil
}
