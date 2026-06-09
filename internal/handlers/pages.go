package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"git.rcsmaine.com/chris/library/internal/auth"
	"git.rcsmaine.com/chris/library/internal/middleware"
	"git.rcsmaine.com/chris/library/internal/models"
	"git.rcsmaine.com/chris/library/internal/theme"
)

// pageContext holds common template data for all page handlers.
type pageContext struct {
	Year                   int
	CSRFToken              string
	IsAdmin                bool
	IsAuthenticated        bool
	IsGuest                bool
	Username               string
	SiteName               string
	SiteTagline            string
	Books                  []models.Book
	Book                   *models.Book
	Items                  []models.WishlistItem
	Users                  []map[string]interface{}
	FamilyMembers          []models.FamilyMember
	ReadingLogs            []models.ReadingLog
	RecentBooks            interface{} // []bookSelect for reading-log page
	Settings               map[string]string
	DefaultGuestVisibility map[string]bool
	CurrentQuery           string
	TotalResults           int
	// Pagination
	Page            int
	PerPage         int
	TotalPages      int
	PaginationStart int
	PaginationEnd   int
	ActiveTheme     theme.Theme
	AvailableThemes []theme.Theme
	ThemeColorsJSON template.HTML // JSON map of theme ID → {bg, text} for switchTheme()

	// Form page fields
	Title            string
	IsEdit           bool
	CancelURL        string
	ActionURL        string
	BookTitle        string
	ItemTitle        string
	Subtitle         string
	Authors          string
	Illustrators     string
	ISBN             string
	Publisher        string
	PublicationYear  string
	PageCount        string
	BookType         string
	Condition        string
	Genres           string
	Themes           string
	Awards           string
	ReadingLevels    string
	GiftFrom         string
	GiftRelationship string
	DateReceived     string
	Location         string
	AgeRange         string
	CoverImageURL    string
	Notes            string
	ChildRating      int
	Quantity         int
	Author           string // for wishlist
	Reason           string // for wishlist
	Priority         int    // for wishlist
	AmazonURL        string // for wishlist
	ThriftbooksURL   string // for wishlist
}

// buildPageContext creates a pageContext for the given request.
// It first checks the request context (set by auth middleware), then falls
// back to reading the session directly for routes without that middleware.
func buildPageContext(r *http.Request, store *sessions.CookieStore, sessionName string, db *sql.DB) pageContext {
	ctx := pageContext{Year: time.Now().Year()}

	// Check context first (set by auth middleware on protected routes).
	if user := auth.GetUserFromContext(r); user != nil {
		ctx.IsAuthenticated = true
		ctx.IsAdmin = !user.IsGuest
		ctx.IsGuest = user.IsGuest
		ctx.Username = user.Username
		if s := middleware.GetSessionFromContext(r); s != nil {
			if token, ok := s.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
				ctx.CSRFToken = token
			}
		} else {
			// Session not in context (e.g. GET request through RequireAuthHTML).
			// Read from store to extract the CSRF token.
			if session, err := store.Get(r, sessionName); err == nil {
				if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
					ctx.CSRFToken = token
				}
			}
		}
		ctx.ActiveTheme = loadActiveTheme(db)
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
		ctx.IsGuest = true
	}

	// Extract CSRF token from session if available.
	if token, ok := session.Values[middleware.CSRFTokenKey].(string); ok && token != "" {
		ctx.CSRFToken = token
	}

	ctx.ActiveTheme = loadActiveTheme(db)

	return ctx
}

// loadActiveTheme reads the active_theme setting and returns the resolved Theme.
// Falls back to WoodlandFairytale if the setting is missing, unknown, or db is nil.
func loadActiveTheme(db *sql.DB) theme.Theme {
	if db == nil {
		return theme.WoodlandFairytale()
	}
	var val string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "active_theme").Scan(&val)
	if err != nil || val == "" {
		return theme.WoodlandFairytale()
	}
	return theme.GetThemeByID(val)
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
		ctx := buildPageContext(r, store, sessionName, db)

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
// Supports ?q= search parameter for filtering books.
func RenderBooksPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName, db)

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

		// Count query
		var total int
		if q != "" {
			like := "%" + q + "%"
			err := db.QueryRow(
				"SELECT COUNT(*) FROM books WHERE title LIKE ? OR authors LIKE ? OR isbn LIKE ? OR genres LIKE ? OR themes LIKE ? OR awards LIKE ? OR reading_levels LIKE ?",
				like, like, like, like, like, like, like,
			).Scan(&total)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
		} else {
			err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
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

		// Data query with LIMIT/OFFSET
		var rows *sql.Rows
		var err error
		if q != "" {
			like := "%" + q + "%"
			rows, err = db.Query(
				"SELECT id, title, authors, isbn, cover_image_url, created_at FROM books WHERE title LIKE ? OR authors LIKE ? OR isbn LIKE ? OR genres LIKE ? OR themes LIKE ? OR awards LIKE ? OR reading_levels LIKE ? ORDER BY title ASC LIMIT ? OFFSET ?",
				like, like, like, like, like, like, like, perPage, offset,
			)
		} else {
			rows, err = db.Query(
				"SELECT id, title, authors, isbn, cover_image_url, created_at FROM books ORDER BY title ASC LIMIT ? OFFSET ?",
				perPage, offset,
			)
		}
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
		ctx := buildPageContext(r, store, sessionName, db)

		// Defense-in-depth: reject unauthenticated requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		id := chi.URLParam(r, "id")

		var book models.Book
		var title, createdAt, updatedAt string
		var subtitle, author, illustrators, publisher, bookType, readingLevels, genres, themes, awards, giftFrom, giftRelationship, dateReceived, condition, location, notes, lastReadDate, coverImage, coverSource, guestVisibleFields sql.NullString
		var pubYear, pageCount, childRating, quantity, readCount sql.NullInt64

		err := db.QueryRow(`SELECT id, title, subtitle, authors, illustrators, isbn, publisher, publication_year, page_count, book_type, reading_levels, genres, themes, awards, gift_from, gift_relationship, date_received, condition, location, notes, child_rating, quantity, read_count, last_read_date, cover_image_url, cover_source, guest_visible_fields, created_at, updated_at FROM books WHERE id = ?`, id).Scan(
			&book.ID, &title, &subtitle, &author, &illustrators, &book.ISBN, &publisher, &pubYear, &pageCount, &bookType, &readingLevels, &genres, &themes, &awards, &giftFrom, &giftRelationship, &dateReceived, &condition, &location, &notes, &childRating, &quantity, &readCount, &lastReadDate, &coverImage, &coverSource, &guestVisibleFields, &createdAt, &updatedAt,
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
		if quantity.Valid {
			book.Quantity = int(quantity.Int64)
		} else {
			book.Quantity = 1
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

// RenderWishlistPage renders the wishlist page (open to guests).
func RenderWishlistPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName, db)

		rows, err := db.Query("SELECT id, isbn, title, author, reason, priority, amazon_url, thriftbooks_url, notes, fulfilled, requested_by, requested_at, fulfilled_at, cover_image_url FROM wishlist ORDER BY priority DESC, requested_at DESC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		items := make([]models.WishlistItem, 0)
		for rows.Next() {
			var item models.WishlistItem
			var isbn, author, reason, amazonURL, thriftbooksURL, notes sql.NullString
			var fulfilled bool
			var requestedBy sql.NullString
			var requestedAtStr, fulfilledAtStr sql.NullString
			var coverImageURL sql.NullString
			if err := rows.Scan(&item.ID, &isbn, &item.Title, &author, &reason, &item.Priority, &amazonURL, &thriftbooksURL, &notes, &fulfilled, &requestedBy, &requestedAtStr, &fulfilledAtStr, &coverImageURL); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			if isbn.Valid {
				item.ISBN = &isbn.String
			}
			if author.Valid {
				item.Author = &author.String
			}
			if reason.Valid {
				item.Reason = &reason.String
			}
			if amazonURL.Valid {
				item.AmazonURL = &amazonURL.String
			}
			if thriftbooksURL.Valid {
				item.ThriftbooksURL = &thriftbooksURL.String
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
			if coverImageURL.Valid {
				item.CoverImageURL = &coverImageURL.String
			}
			item.Fulfilled = fulfilled
			if fulfilledAtStr.Valid {
				item.FulfilledAt = &fulfilledAtStr.String
				item.Fulfilled = true
			}
			item.IsAdmin = ctx.IsAdmin
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

// RenderWishlistFormPage renders the add/edit wishlist item form as a full page (admin only).
func RenderWishlistFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, itemID int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName, db)

		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !ctx.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		var item models.WishlistItem
		itemTitle := ""
		cancelURL := "/wishlist"

		if isEdit {
			row := db.QueryRow(`SELECT `+wishlistColumns+` FROM wishlist WHERE id = ?`, itemID)
			itm, err := scanWishlistItem(row)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			item = *itm
			itemTitle = item.Title
			cancelURL = "/wishlist"
		}

		data := pageContext{
			Year:            ctx.Year,
			CSRFToken:       ctx.CSRFToken,
			IsAdmin:         ctx.IsAdmin,
			IsAuthenticated: ctx.IsAuthenticated,
			Username:        ctx.Username,
			ActiveTheme:     ctx.ActiveTheme,
			SiteName:        ctx.SiteName,
			SiteTagline:     ctx.SiteTagline,
			AvailableThemes: ctx.AvailableThemes,
			IsEdit:          isEdit,
			ItemTitle:       itemTitle,
			CancelURL:       cancelURL,
			ActionURL: func() string {
				if isEdit {
					return "/wishlist/" + strconv.FormatInt(itemID, 10) + "/update"
				}
				return "/wishlist/create"
			}(),
			Title:  item.Title,
			Author: derefString(item.Author),
			ISBN:   derefString(item.ISBN),
			Reason: derefString(item.Reason),
			Priority: func() int {
				if isEdit {
					return item.Priority
				}
				return 3
			}(),
			AmazonURL:      derefString(item.AmazonURL),
			ThriftbooksURL: derefString(item.ThriftbooksURL),
			CoverImageURL:  derefString(item.CoverImageURL),
			Notes:          derefString(item.Notes),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "wishlist-form.html", data); err != nil {
			slog.Error("template error", "page", "wishlist-form", "error", err)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// RenderSettingsPage renders the settings page (admin required).
func RenderSettingsPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName, db)

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

		// Load family members
		fmRows, err := db.Query("SELECT id, name, relation, created_at, updated_at FROM family_members ORDER BY name ASC")
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer fmRows.Close()

		familyMembers := make([]models.FamilyMember, 0)
		for fmRows.Next() {
			var fm models.FamilyMember
			var createdAt, updatedAt string
			if err := fmRows.Scan(&fm.ID, &fm.Name, &fm.Relation, &createdAt, &updatedAt); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			fm.CreatedAt = createdAt
			fm.UpdatedAt = updatedAt
			familyMembers = append(familyMembers, fm)
		}
		if err = fmRows.Err(); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		ctx.FamilyMembers = familyMembers

		// Load default guest visibility
		var defaultVisibility string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "default_guest_visibility").Scan(&defaultVisibility); err == nil {
			ctx.DefaultGuestVisibility = make(map[string]bool)
			_ = json.Unmarshal([]byte(defaultVisibility), &ctx.DefaultGuestVisibility)
		}

		// Load available themes
		ctx.AvailableThemes = theme.AvailableThemes()

		// Build theme colors map for server-side JS rendering
		ctx.ThemeColorsJSON = buildThemeColorsJSON(theme.AvailableThemes())

		renderPage(w, r, tmpl, "settings.html", ctx)
	}
}

// RenderBookFormPage renders the add/edit book form as a full page.
func RenderBookFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, bookID int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := buildPageContext(r, store, sessionName, db)

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

		data := pageContext{
			Year:            ctx.Year,
			CSRFToken:       ctx.CSRFToken,
			IsAdmin:         ctx.IsAdmin,
			IsAuthenticated: ctx.IsAuthenticated,
			Username:        ctx.Username,
			ActiveTheme:     ctx.ActiveTheme,
			SiteName:        ctx.SiteName,
			SiteTagline:     ctx.SiteTagline,
			AvailableThemes: ctx.AvailableThemes,
			IsEdit:          isEdit,
			BookTitle:       bookTitle,
			CancelURL:       cancelURL,
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
			BookType:         derefString(book.BookType),
			Condition:        derefString(book.Condition),
			Genres:           derefString(book.Genres),
			Themes:           derefString(book.Themes),
			Awards:           derefString(book.Awards),
			ReadingLevels:    derefString(book.ReadingLevels),
			GiftFrom:         derefString(book.GiftFrom),
			GiftRelationship: derefString(book.GiftRelationship),
			DateReceived:     derefString(book.DateReceived),
			Location:         derefString(book.Location),
			AgeRange:         derefString(book.AgeRange),
			CoverImageURL:    derefString(book.CoverImageURL),
			Notes:            derefString(book.Notes),
			ChildRating:      derefInt(book.ChildRating),
			Quantity: func() int {
				if book.Quantity > 0 {
					return book.Quantity
				}
				return 1
			}(),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "book-form.html", data); err != nil {
			slog.Error("template error", "page", "book-form", "error", err)
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// --- Exported for testing ---

// PageContextForTest is the exported pageContext struct for test access.
type PageContextForTest struct {
	CSRFToken       string
	IsAdmin         bool
	IsAuthenticated bool
	Username        string
	ActiveTheme     theme.Theme
}

// BuildPageContextForTest calls buildPageContext and returns the fields
// needed for CSRF token verification and theme loading in tests.
func BuildPageContextForTest(r *http.Request, store *sessions.CookieStore, sessionName string) PageContextForTest {
	ctx := buildPageContext(r, store, sessionName, nil)
	return PageContextForTest{
		CSRFToken:       ctx.CSRFToken,
		IsAdmin:         ctx.IsAdmin,
		IsAuthenticated: ctx.IsAuthenticated,
		Username:        ctx.Username,
		ActiveTheme:     ctx.ActiveTheme,
	}
}

// buildThemeColorsJSON builds a JSON map of theme ID → {bg, text}
// from the available themes, for server-side rendering into the
// switchTheme() JS in settings.html.
func buildThemeColorsJSON(themes []theme.Theme) template.HTML {
	result := "{"
	for i, t := range themes {
		if i > 0 {
			result += ","
		}
		// #nosec G203 -- values come from application-controlled theme definitions, not user input
		result += fmt.Sprintf(`"%s":{"bg":"%s","text":"%s"}`, t.ID, t.Background, t.Text)
	}
	result += "}"
	// #nosec G203 -- values come from application-controlled theme definitions
	return template.HTML(result)
}
