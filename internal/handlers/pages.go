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

	"github.com/Toph4er/family-library/internal/auth"
	sqldb "github.com/Toph4er/family-library/internal/db"
	"github.com/Toph4er/family-library/internal/middleware"
	"github.com/Toph4er/family-library/internal/models"
	"github.com/Toph4er/family-library/internal/theme"
)

// BaseContext holds common template data shared across all page handlers.
type BaseContext struct {
	Year            int
	CSRFToken       string
	Nonce           string
	IsAdmin         bool
	IsAuthenticated bool
	IsGuest         bool
	Username        string
	SiteName        string
	SiteTagline     string
	ActiveTheme     theme.Theme
	AvailableThemes []theme.Theme
	ThemeColorsJSON template.HTML
}

// PaginationContext holds pagination data for list pages.
type PaginationContext struct {
	Page            int
	PerPage         int
	TotalPages      int
	PaginationStart int
	PaginationEnd   int
}

// BookListContext holds data for the books listing page.
type BookListContext struct {
	Books        []models.Book
	CurrentQuery string
	TotalResults int
}

// BookDetailContext holds data for the book detail page.
type BookDetailContext struct {
	Book *models.Book
}

// BookFormContext holds data for the add/edit book form page.
type BookFormContext struct {
	BookID            int64
	IsEdit            bool
	BookTitle         string
	CancelURL         string
	ActionURL         string
	Title             string
	Subtitle          string
	Authors           string
	Illustrators      string
	ISBN              string
	Publisher         string
	PublicationYear   string
	PageCount         string
	BookType          string
	Condition         string
	Genres            string
	Themes            string
	Awards            string
	ReadingLevels     string
	DeweyDecimalClass string
	Language          string
	Series            string
	AgeRange          string
	SubjectPlaces     string
	SubjectPeople     string
	SubjectTimes      string
	Description       string
	GiftFrom          string
	GiftRelationship  string
	DateReceived      string
	Location          string
	CoverImageURL     string
	Notes             string
	ChildRating       int
	Quantity          int
}

// WishlistListContext holds data for the wishlist listing page.
type WishlistListContext struct {
	Items []models.WishlistItem
}

// WishlistFormContext holds data for the add/edit wishlist item form page.
type WishlistFormContext struct {
	IsWishlistEdit        bool
	ItemTitle             string
	WishlistCancelURL     string
	WishlistActionURL     string
	WishlistTitle         string
	Author                string
	WishlistISBN          string
	Reason                string
	Priority              int
	AmazonURL             string
	ThriftbooksURL        string
	WishlistCoverImageURL string
	WishlistNotes         string
}

// FamilyMembersContext holds family member data (shared by settings and reading-log pages).
type FamilyMembersContext struct {
	FamilyMembers []models.FamilyMember
}

// SettingsContext holds data for the settings page.
type SettingsContext struct {
	Settings               map[string]string
	Users                  []map[string]interface{}
	DefaultGuestVisibility map[string]bool
}

// ReadingLogContext holds data for the reading log page.
type ReadingLogContext struct {
	ReadingLogs []models.ReadingLog
	RecentBooks interface{} // []bookSelect
}

// StatCard represents a top-of-page stat card (big number).
type StatCard struct {
	Icon  template.HTML
	Value string
	Label string
	Link  string
}

// SectionCard represents a side-by-side info panel.
type SectionCard struct {
	Icon  template.HTML
	Title string
	Rows  []SectionRow
	Link  string
}

// SectionRow is a single label/value pair inside a SectionCard.
type SectionRow struct {
	Label string
	Value string
	Link  string
}

// ActivityEntry is a single line in the recent activity panel.
type ActivityEntry struct {
	Text string
	Time string
	Link string
}

// DashboardContext holds data for the dashboard page.
type DashboardContext struct {
	StatCards         []StatCard
	Sections          []SectionCard
	CollectionStats   []SectionRow
	Activity          []ActivityEntry
	ReaderBreakdown   []ReaderBreakdownRow
	GenreBreakdown    []GenreBreakdownRow
	BookTypeBreakdown []BookTypeBreakdownRow
}

// ReaderBreakdownRow is a single row in the "Reading by Reader" panel.
type ReaderBreakdownRow struct {
	Reader string
	Count  int
	Width  string // CSS width percentage
}

// GenreBreakdownRow is a single row in the "Genre Breakdown" panel.
type GenreBreakdownRow struct {
	Genre string
	Count int
	Width string // CSS width percentage
}

// BookTypeBreakdownRow is a single row in the "Books by Book Type" panel.
type BookTypeBreakdownRow struct {
	BookType string
	Count    int
	Width    string // CSS width percentage
}

// pageContext is the composite context used by all page handlers.
// It embeds smaller context structs so template access via {{.FieldName}}
// continues to work transparently.
type pageContext struct {
	BaseContext
	PaginationContext
	BookListContext
	BookDetailContext
	BookFormContext
	WishlistListContext
	WishlistFormContext
	FamilyMembersContext
	SettingsContext
	ReadingLogContext
	DashboardContext
}

// buildBaseContext creates a BaseContext for the given request.
// It first checks the request context (set by auth middleware), then falls
// back to reading the session directly for routes without that middleware.
func buildBaseContext(r *http.Request, store *sessions.CookieStore, sessionName string, db *sql.DB) BaseContext {
	ctx := BaseContext{Year: time.Now().Year()}
	ctx.Nonce = middleware.GetCSPNonce(r)

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
			HTMXError(w, http.StatusInternalServerError)
		}
	} else {
		// Full page request — render the page template (which includes base).
		if err := tmpl.ExecuteTemplate(w, pageName, data); err != nil {
			slog.Error("template error", "page", pageName, "error", err)
			HTMXError(w, http.StatusInternalServerError)
		}
	}
}

// RenderLandingPage renders the public landing page.
// Authenticated users are redirected to /books.
func RenderLandingPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		// Authenticated users should not see the landing page — send them to dashboard.
		if ctx.IsAuthenticated {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
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

// RenderDashboardPage renders the dashboard page (auth required).
func RenderDashboardPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		dash := DashboardContext{}

		// --- Stat cards ---

		// Total books
		var bookCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&bookCount); err == nil {
			dash.StatCards = append(dash.StatCards, StatCard{
				Icon:  svgIcon("book"),
				Value: strconv.Itoa(bookCount),
				Label: "Total Books",
				Link:  "/books",
			})
		}

		// Total reads
		var totalReads int
		if err := db.QueryRow("SELECT COUNT(*) FROM reading_logs").Scan(&totalReads); err == nil {
			dash.StatCards = append(dash.StatCards, StatCard{
				Icon:  svgIcon("open-book"),
				Value: strconv.Itoa(totalReads),
				Label: "Total Reads",
				Link:  "/reading-log",
			})
		}

		// Average child rating (among rated books)
		var avgRating float64
		if err := db.QueryRow("SELECT COALESCE(AVG(child_rating), 0) FROM books WHERE child_rating IS NOT NULL AND child_rating > 0").Scan(&avgRating); err == nil {
			dash.StatCards = append(dash.StatCards, StatCard{
				Icon:  svgIcon("star"),
				Value: fmt.Sprintf("%.1f", avgRating),
				Label: "Avg Rating",
				Link:  "/books",
			})
		}

		// Wishlist items (unfulfilled)
		var wishlistCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM wishlist WHERE NOT fulfilled").Scan(&wishlistCount); err == nil {
			dash.StatCards = append(dash.StatCards, StatCard{
				Icon:  svgIcon("clipboard-list"),
				Value: strconv.Itoa(wishlistCount),
				Label: "Wishlist Items",
				Link:  "/wishlist",
			})
		}

		// --- Section: Top 5 Most Read ---
		var mostReadRows []SectionRow
		rows, err := db.Query(`
			SELECT b.title, COUNT(rl.id) AS read_count
			FROM reading_logs rl
			JOIN books b ON b.id = rl.book_id
			GROUP BY rl.book_id
			ORDER BY read_count DESC
			LIMIT 5
		`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var title string
				var count int
				if err := rows.Scan(&title, &count); err == nil {
					mostReadRows = append(mostReadRows, SectionRow{
						Label: title,
						Value: fmt.Sprintf("%d read%s", count, pluralS(count, "s", "")),
						Link:  fmt.Sprintf("/books/%d", 0), // will be fixed below
					})
				}
			}
			// Fix: get book IDs for links
			rows2, err2 := db.Query(`
				SELECT b.id, b.title, COUNT(rl.id) AS read_count
				FROM reading_logs rl
				JOIN books b ON b.id = rl.book_id
				GROUP BY rl.book_id
				ORDER BY read_count DESC
				LIMIT 5
			`)
			if err2 == nil {
				mostReadRows = nil
				for rows2.Next() {
					var id int64
					var title string
					var count int
					if err := rows2.Scan(&id, &title, &count); err == nil {
						mostReadRows = append(mostReadRows, SectionRow{
							Label: title,
							Value: fmt.Sprintf("%d read%s", count, pluralS(count, "s", "")),
							Link:  fmt.Sprintf("/books/%d", id),
						})
					}
				}
				rows2.Close()
			}
		}
		dash.Sections = append(dash.Sections, SectionCard{
			Icon:  svgIcon("open-book"),
			Title: "Most Read Books",
			Rows:  mostReadRows,
			Link:  "/reading-log",
		})

		// --- Section: Top 5 Highest Rated ---
		var topRatedRows []SectionRow
		rows3, err := db.Query(`
			SELECT title, child_rating
			FROM books
			WHERE child_rating IS NOT NULL AND child_rating > 0
			ORDER BY child_rating DESC
			LIMIT 5
		`)
		if err == nil {
			defer rows3.Close()
			for rows3.Next() {
				var title string
				var rating int
				if err := rows3.Scan(&title, &rating); err == nil {
					topRatedRows = append(topRatedRows, SectionRow{
						Label: title,
						Value: fmt.Sprintf("%d/5.0", rating),
					})
				}
			}
		}
		dash.Sections = append(dash.Sections, SectionCard{
			Icon:  svgIcon("star"),
			Title: "Highest Rated",
			Rows:  topRatedRows,
			Link:  "/books",
		})

		// --- Section: 5 Most Recent Additions ---
		var recentRows []SectionRow
		rows4, err := db.Query(`
			SELECT title, created_at
			FROM books
			ORDER BY created_at DESC
			LIMIT 5
		`)
		if err == nil {
			defer rows4.Close()
			for rows4.Next() {
				var title, createdAt string
				if err := rows4.Scan(&title, &createdAt); err == nil {
					recentRows = append(recentRows, SectionRow{
						Label: title,
						Value: formatDate(createdAt),
					})
				}
			}
		}
		dash.Sections = append(dash.Sections, SectionCard{
			Icon:  svgIcon("sparkles"),
			Title: "Recently Added",
			Rows:  recentRows,
			Link:  "/books",
		})

		// --- Section: 5 Most Wanted (wishlist) ---
		var wishlistRows []SectionRow
		rows5, err := db.Query(`
			SELECT title, author, priority
			FROM wishlist
			WHERE NOT fulfilled
			ORDER BY priority ASC
			LIMIT 5
		`)
		if err == nil {
			defer rows5.Close()
			for rows5.Next() {
				var title, author string
				var priority int
				if err := rows5.Scan(&title, &author, &priority); err == nil {
					label := title
					if author != "" {
						label = title + " — " + author
					}
					wishlistRows = append(wishlistRows, SectionRow{
						Label: label,
						Value: fmt.Sprintf("Priority %d", priority),
					})
				}
			}
		}
		dash.Sections = append(dash.Sections, SectionCard{
			Icon:  svgIcon("clipboard-list"),
			Title: "Most Wanted",
			Rows:  wishlistRows,
			Link:  "/wishlist",
		})

		// --- Section: Reading by Reader ---
		var readerRows []ReaderBreakdownRow
		rows6, err := db.Query(`
			SELECT reader_name, COUNT(*) AS cnt
			FROM reading_logs
			GROUP BY reader_name
			ORDER BY cnt DESC
		`)
		if err == nil {
			defer rows6.Close()
			for rows6.Next() {
				var reader string
				var count int
				if err := rows6.Scan(&reader, &count); err == nil {
					readerRows = append(readerRows, ReaderBreakdownRow{
						Reader: reader,
						Count:  count,
					})
				}
			}
		}
		// Calculate widths for reader bars
		if len(readerRows) > 0 {
			var maxCount int
			for _, r := range readerRows {
				if r.Count > maxCount {
					maxCount = r.Count
				}
			}
			for i := range readerRows {
				if maxCount > 0 {
					readerRows[i].Width = fmt.Sprintf("%.0f%%", float64(readerRows[i].Count)/float64(maxCount)*100)
				} else {
					readerRows[i].Width = "0%"
				}
			}
		}
		dash.ReaderBreakdown = readerRows

		// --- Section: Genre Breakdown (using json_each) ---
		var genreRows []GenreBreakdownRow
		rows7, err := db.Query(`
			SELECT genre, COUNT(*) AS cnt
			FROM books, json_each(books.genres)
			GROUP BY genre
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer rows7.Close()
			for rows7.Next() {
				var genre string
				var count int
				if err := rows7.Scan(&genre, &count); err == nil {
					genreRows = append(genreRows, GenreBreakdownRow{
						Genre: genre,
						Count: count,
					})
				}
			}
		}
		dash.GenreBreakdown = genreRows
		// Calculate widths for genre bars
		if len(genreRows) > 0 {
			var maxCount int
			for _, g := range genreRows {
				if g.Count > maxCount {
					maxCount = g.Count
				}
			}
			for i := range genreRows {
				if maxCount > 0 {
					genreRows[i].Width = fmt.Sprintf("%.0f%%", float64(genreRows[i].Count)/float64(maxCount)*100)
				} else {
					genreRows[i].Width = "0%"
				}
			}
		}

		// --- Section: Books by Book Type ---
		var bookTypeRows []BookTypeBreakdownRow
		rows8, err := db.Query(`
			SELECT book_type, COUNT(*) AS cnt
			FROM books
			WHERE book_type IS NOT NULL
			GROUP BY book_type
			ORDER BY cnt DESC
		`)
		if err == nil {
			defer rows8.Close()
			for rows8.Next() {
				var bookType string
				var count int
				if err := rows8.Scan(&bookType, &count); err == nil {
					bookTypeRows = append(bookTypeRows, BookTypeBreakdownRow{
						BookType: bookType,
						Count:    count,
					})
				}
			}
		}
		dash.BookTypeBreakdown = bookTypeRows

		// --- Section: Collection Stats ---
		var firstBookDate, lastReadDate string
		if err := db.QueryRow("SELECT MIN(created_at) FROM books").Scan(&firstBookDate); err != nil {
			firstBookDate = ""
		}
		if err := db.QueryRow("SELECT MAX(read_at) FROM reading_logs").Scan(&lastReadDate); err != nil {
			lastReadDate = ""
		}

		var totalPages int
		if err := db.QueryRow("SELECT COALESCE(SUM(page_count), 0) FROM books WHERE page_count IS NOT NULL").Scan(&totalPages); err != nil {
			totalPages = 0
		}

		var coversUploaded int
		if err := db.QueryRow("SELECT COUNT(*) FROM books WHERE cover_image_url IS NOT NULL").Scan(&coversUploaded); err != nil {
			coversUploaded = 0
		}

		var booksReadCount int
		if err := db.QueryRow("SELECT COUNT(DISTINCT book_id) FROM reading_logs").Scan(&booksReadCount); err != nil {
			booksReadCount = 0
		}

		var booksRatedCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM books WHERE child_rating IS NOT NULL AND child_rating >= 4").Scan(&booksRatedCount); err != nil {
			booksRatedCount = 0
		}

		dash.CollectionStats = []SectionRow{
			{Label: "First book added", Value: formatDate(firstBookDate)},
			{Label: "Last read date", Value: formatDate(lastReadDate)},
			{Label: "Total pages cataloged", Value: formatNumber(totalPages) + " pages"},
			{Label: "Covers uploaded", Value: fmt.Sprintf("%d/%d", coversUploaded, bookCount)},
			{Label: "Books read ≥1 time", Value: fmt.Sprintf("%d/%d", booksReadCount, bookCount)},
			{Label: "Books rated ≥4★", Value: fmt.Sprintf("%d/%d", booksRatedCount, bookCount)},
		}

		// --- Section: Books Read This Month ---
		var booksReadThisMonth int
		now := time.Now()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0)
		err = db.QueryRow(
			"SELECT COUNT(DISTINCT book_id) FROM reading_logs WHERE read_at >= ? AND read_at < ?",
			monthStart.Format("2006-01-02 15:04:05"),
			monthEnd.Format("2006-01-02 15:04:05"),
		).Scan(&booksReadThisMonth)
		if err == nil {
			dash.StatCards = append(dash.StatCards, StatCard{
				Icon:  svgIcon("calendar"),
				Value: strconv.Itoa(booksReadThisMonth),
				Label: "Read This Month",
				Link:  "/reading-log",
			})
		}

		ctx.DashboardContext = dash

		renderPage(w, r, tmpl, "dashboard.html", ctx)
	}
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
				q,
			).Scan(&total)
			if err != nil {
				HTMXError(w, http.StatusInternalServerError)
				return
			}
		} else {
			err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)
			if err != nil {
				HTMXError(w, http.StatusInternalServerError)
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
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		books := make([]models.Book, 0)
		for rows.Next() {
			var book models.Book
			var author, isbn, coverImage sql.NullString
			var createdAt string
			if err := rows.Scan(&book.ID, &book.Title, &author, &isbn, &coverImage, &createdAt); err != nil {
				HTMXError(w, http.StatusInternalServerError)
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
			HTMXError(w, http.StatusInternalServerError)
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
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		book = *b

		ctx.Book = &book
		filterBookForGuest(r, &book)

		renderPage(w, r, tmpl, "book-detail.html", ctx)
	}
}

// RenderWishlistPage renders the wishlist page (open to guests).
func RenderWishlistPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		rows, err := db.Query("SELECT id, isbn, title, author, reason, priority, amazon_url, thriftbooks_url, notes, fulfilled, requested_by, requested_at, fulfilled_at, cover_image_url FROM wishlist ORDER BY priority DESC, requested_at DESC")
		if err != nil {
			HTMXError(w, http.StatusInternalServerError)
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
				HTMXError(w, http.StatusInternalServerError)
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
			HTMXError(w, http.StatusInternalServerError)
			return
		}

		ctx.Items = items

		renderPage(w, r, tmpl, "wishlist.html", ctx)
	}
}

// RenderWishlistFormPage renders the add/edit wishlist item form as a full page (admin only).
func RenderWishlistFormPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string, isEdit bool, itemID int64) http.HandlerFunc {
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

		var item models.WishlistItem
		itemTitle := ""
		cancelURL := "/wishlist"

		if isEdit {
			row := db.QueryRow(`SELECT `+wishlistColumns+` FROM wishlist WHERE id = ?`, itemID)
			itm, err := sqldb.ScanWishlistItem(row)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			item = *itm
			itemTitle = item.Title
			cancelURL = "/wishlist"
		}

		data := pageContext{
			BaseContext: ctx.BaseContext,
			WishlistFormContext: WishlistFormContext{
				IsWishlistEdit:    isEdit,
				ItemTitle:         itemTitle,
				WishlistCancelURL: cancelURL,
				WishlistActionURL: func() string {
					if isEdit {
						return "/wishlist/" + strconv.FormatInt(itemID, 10) + "/update"
					}
					return "/wishlist/create"
				}(),
				WishlistTitle: item.Title,
				Author:        derefString(item.Author),
				WishlistISBN:  derefString(item.ISBN),
				Reason:        derefString(item.Reason),
				Priority: func() int {
					if isEdit {
						return item.Priority
					}
					return 3
				}(),
				AmazonURL:             derefString(item.AmazonURL),
				ThriftbooksURL:        derefString(item.ThriftbooksURL),
				WishlistCoverImageURL: derefString(item.CoverImageURL),
				WishlistNotes:         derefString(item.Notes),
			},
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "wishlist-form.html", data); err != nil {
			slog.Error("template error", "page", "wishlist-form", "error", err)
			HTMXError(w, http.StatusInternalServerError)
		}
	}
}

// RenderSettingsPage renders the settings page (admin required).
func RenderSettingsPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

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
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				HTMXError(w, http.StatusInternalServerError)
				return
			}
			if _, sensitive := sensitiveKeys[key]; sensitive {
				continue
			}
			settings[key] = value
		}
		if err = rows.Err(); err != nil {
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		ctx.Settings = settings

		// Load users
		userRows, err := db.Query("SELECT id, username, role, display_name, created_at FROM users ORDER BY id")
		if err != nil {
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		defer userRows.Close()

		users := make([]map[string]interface{}, 0)
		for userRows.Next() {
			var id int64
			var username, role, displayName, createdAt string
			if err := userRows.Scan(&id, &username, &role, &displayName, &createdAt); err != nil {
				HTMXError(w, http.StatusInternalServerError)
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
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		ctx.Users = users

		// Load family members
		fmRows, err := db.Query("SELECT id, name, relation, created_at, updated_at FROM family_members ORDER BY name ASC")
		if err != nil {
			HTMXError(w, http.StatusInternalServerError)
			return
		}
		defer fmRows.Close()

		familyMembers := make([]models.FamilyMember, 0)
		for fmRows.Next() {
			var fm models.FamilyMember
			var createdAt, updatedAt string
			if err := fmRows.Scan(&fm.ID, &fm.Name, &fm.Relation, &createdAt, &updatedAt); err != nil {
				HTMXError(w, http.StatusInternalServerError)
				return
			}
			fm.CreatedAt = createdAt
			fm.UpdatedAt = updatedAt
			familyMembers = append(familyMembers, fm)
		}
		if err = fmRows.Err(); err != nil {
			HTMXError(w, http.StatusInternalServerError)
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
			slog.Error("template error", "page", "book-form", "error", err)
			HTMXError(w, http.StatusInternalServerError)
		}
	}
}

// --- Exported for testing ---

// PageContextForTest is the exported BaseContext struct for test access.
type PageContextForTest = BaseContext

// BuildPageContextForTest calls buildBaseContext and returns the context
// needed for CSRF token verification and theme loading in tests.
func BuildPageContextForTest(r *http.Request, store *sessions.CookieStore, sessionName string) PageContextForTest {
	return buildBaseContext(r, store, sessionName, nil)
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

// --- Dashboard helpers ---

// svgIcon returns a 24x24 inline SVG as template.HTML.
func svgIcon(name string) template.HTML {
	const (
		iconBook          = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>`
		iconOpenBook      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/></svg>`
		iconStar          = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1" aria-hidden="true"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>`
		iconClipboardList = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="8" y="2" width="8" height="4" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M9 14l2 2 4-4"/></svg>`
		iconCalendar      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>`
		iconUsers         = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`
		iconTag           = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>`
		iconBarChart      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><line x1="12" y1="20" x2="12" y2="10"/><line x1="18" y1="20" x2="18" y2="4"/><line x1="6" y1="20" x2="6" y2="16"/></svg>`
		iconHeart         = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>`
		iconSparkles      = `<svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z"/></svg>`
	)
	switch name {
	case "book":
		return template.HTML(iconBook) // #nosec G203 -- hardcoded SVG, not user input
	case "open-book":
		return template.HTML(iconOpenBook) // #nosec G203 -- hardcoded SVG, not user input
	case "star":
		return template.HTML(iconStar) // #nosec G203 -- hardcoded SVG, not user input
	case "clipboard-list":
		return template.HTML(iconClipboardList) // #nosec G203 -- hardcoded SVG, not user input
	case "calendar":
		return template.HTML(iconCalendar) // #nosec G203 -- hardcoded SVG, not user input
	case "users":
		return template.HTML(iconUsers) // #nosec G203 -- hardcoded SVG, not user input
	case "tag":
		return template.HTML(iconTag) // #nosec G203 -- hardcoded SVG, not user input
	case "bar-chart":
		return template.HTML(iconBarChart) // #nosec G203 -- hardcoded SVG, not user input
	case "heart":
		return template.HTML(iconHeart) // #nosec G203 -- hardcoded SVG, not user input
	case "sparkles":
		return template.HTML(iconSparkles) // #nosec G203 -- hardcoded SVG, not user input
	default:
		return ""
	}
}

func pluralS(n int, plural, singular string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func formatDate(s string) string {
	if s == "" {
		return "—"
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return s
	}
	return t.Format("Jan 2, 2006")
}

func formatNumber(n int) string {
	if n < 0 {
		return "0"
	}
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%d", n)
	}
	return strconv.Itoa(n)
}
