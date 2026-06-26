// Package services provides domain-level business logic, decoupled from HTTP handling.
package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// StatCard represents a top-of-page stat card (big number).
type StatCard struct {
	Value string
	Label string
	Link  string
}

// SectionRow is a single label/value pair inside a dashboard section.
type SectionRow struct {
	Label string
	Value string
	Link  string
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

// DashboardData holds all data needed to render the dashboard page.
type DashboardData struct {
	StatCards         []StatCard
	CollectionStats   []SectionRow
	MostRead          []SectionRow
	HighestRated      []SectionRow
	RecentlyAdded     []SectionRow
	MostWanted        []SectionRow
	ReaderBreakdown   []ReaderBreakdownRow
	GenreBreakdown    []GenreBreakdownRow
	BookTypeBreakdown []BookTypeBreakdownRow
}

// DashboardService handles all dashboard data queries.
type DashboardService struct {
	db *sql.DB
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{db: db}
}

// Get returns all dashboard data in a single call.
func (s *DashboardService) Get(ctx context.Context) (*DashboardData, error) {
	data := &DashboardData{}

	// --- Stat cards ---

	var bookCount int
	if err := queryRowContext(ctx, s.db, "SELECT COUNT(*) FROM books", &bookCount); err != nil {
		return nil, fmt.Errorf("count books: %w", err)
	}

	data.StatCards = append(data.StatCards, StatCard{
		Value: strconv.Itoa(bookCount),
		Label: "Total Books",
		Link:  "/books",
	})

	var totalReads int
	if err := queryRowContext(ctx, s.db, "SELECT COUNT(*) FROM reading_logs", &totalReads); err != nil {
		return nil, fmt.Errorf("count reads: %w", err)
	}

	data.StatCards = append(data.StatCards, StatCard{
		Value: strconv.Itoa(totalReads),
		Label: "Total Reads",
		Link:  "/reading-log",
	})

	var avgRating float64
	if err := queryRowContext(ctx, s.db,
		"SELECT COALESCE(AVG(child_rating), 0) FROM books WHERE child_rating IS NOT NULL AND child_rating > 0", &avgRating); err != nil {
		return nil, fmt.Errorf("avg rating: %w", err)
	}

	data.StatCards = append(data.StatCards, StatCard{
		Value: fmt.Sprintf("%.1f", avgRating),
		Label: "Avg Rating",
		Link:  "/books",
	})

	var wishlistCount int
	if err := queryRowContext(ctx, s.db, "SELECT COUNT(*) FROM wishlist WHERE NOT fulfilled", &wishlistCount); err != nil {
		return nil, fmt.Errorf("count wishlist: %w", err)
	}

	data.StatCards = append(data.StatCards, StatCard{
		Value: strconv.Itoa(wishlistCount),
		Label: "Wishlist Items",
		Link:  "/wishlist",
	})

	// --- Most Read (fixed: one query instead of two) ---
	rows, err := s.db.QueryContext(ctx, `
		SELECT b.id, b.title, COUNT(rl.id) AS read_count
		FROM reading_logs rl
		JOIN books b ON b.id = rl.book_id
		GROUP BY rl.book_id
		ORDER BY read_count DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("most read: %w", err)
	}

	var mostRead []SectionRow
	for rows.Next() {
		var id int64
		var title string
		var count int
		if err := rows.Scan(&id, &title, &count); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan most read: %w", err)
		}
		mostRead = append(mostRead, SectionRow{
			Label: title,
			Value: pluralizeReads(count),
			Link:  fmt.Sprintf("/books/%d", id),
		})
	}
	rows.Close()
	data.MostRead = mostRead

	// --- Highest Rated ---
	rows, err = s.db.QueryContext(ctx, `
		SELECT title, child_rating FROM books
		WHERE child_rating IS NOT NULL AND child_rating > 0
		ORDER BY child_rating DESC LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("highest rated: %w", err)
	}

	var topRated []SectionRow
	for rows.Next() {
		var title string
		var rating int
		if err := rows.Scan(&title, &rating); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan highest rated: %w", err)
		}
		topRated = append(topRated, SectionRow{
			Label: title,
			Value: fmt.Sprintf("%d/5.0", rating),
		})
	}
	rows.Close()
	data.HighestRated = topRated

	// --- Recently Added ---
	rows, err = s.db.QueryContext(ctx, `
		SELECT title, created_at FROM books ORDER BY created_at DESC LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("recently added: %w", err)
	}

	var recent []SectionRow
	for rows.Next() {
		var title string
		var createdAt string
		if err := rows.Scan(&title, &createdAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan recently added: %w", err)
		}
		recent = append(recent, SectionRow{
			Label: title,
			Value: parseDate(createdAt),
		})
	}
	rows.Close()
	data.RecentlyAdded = recent

	// --- Most Wanted (wishlist) ---
	rows, err = s.db.QueryContext(ctx, `
		SELECT title, author, priority FROM wishlist
		WHERE NOT fulfilled ORDER BY priority ASC LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("most wanted: %w", err)
	}

	var mostWanted []SectionRow
	for rows.Next() {
		var title string
		var author sql.NullString
		var priority int
		if err := rows.Scan(&title, &author, &priority); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan most wanted: %w", err)
		}
		label := title
		if author.Valid && author.String != "" {
			label = title + " — " + author.String
		}
		mostWanted = append(mostWanted, SectionRow{
			Label: label,
			Value: fmt.Sprintf("Priority %d", priority),
		})
	}
	rows.Close()
	data.MostWanted = mostWanted

	// --- Reading by Reader ---
	rows, err = s.db.QueryContext(ctx, `
		SELECT reader_name, COUNT(*) AS cnt FROM reading_logs
		GROUP BY reader_name ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("reader breakdown: %w", err)
	}

	var readerRows []ReaderBreakdownRow
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan readers: %w", err)
		}
		readerRows = append(readerRows, ReaderBreakdownRow{Reader: name, Count: count})
	}
	rows.Close()

	if len(readerRows) > 0 {
		maxCount := readerRows[0].Count
		for _, r := range readerRows {
			if r.Count > maxCount {
				maxCount = r.Count
			}
		}
		for i := range readerRows {
			readerRows[i].Width = barWidth(readerRows[i].Count, maxCount)
		}
	}
	data.ReaderBreakdown = readerRows

	// --- Genre Breakdown (best-effort; skip if JSON is malformed) ---
	genreRows, _ := s.getGenreBreakdown(ctx) // ignore error — don't fail dashboard for bad data
	data.GenreBreakdown = genreRows

	if len(genreRows) > 0 {
		maxCount := genreRows[0].Count
		for _, g := range genreRows {
			if g.Count > maxCount {
				maxCount = g.Count
			}
		}
		for i := range genreRows {
			genreRows[i].Width = barWidth(genreRows[i].Count, maxCount)
		}
	}

	// --- Books by Book Type ---
	rows, err = s.db.QueryContext(ctx, `
		SELECT book_type, COUNT(*) AS cnt FROM books
		WHERE book_type IS NOT NULL GROUP BY book_type ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("book type breakdown: %w", err)
	}

	var bookTypeRows []BookTypeBreakdownRow
	for rows.Next() {
		var bt string
		var count int
		if err := rows.Scan(&bt, &count); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan book types: %w", err)
		}
		bookTypeRows = append(bookTypeRows, BookTypeBreakdownRow{BookType: bt, Count: count})
	}
	rows.Close()

	if len(bookTypeRows) > 0 {
		maxCount := bookTypeRows[0].Count
		for _, b := range bookTypeRows {
			if b.Count > maxCount {
				maxCount = b.Count
			}
		}
		for i := range bookTypeRows {
			bookTypeRows[i].Width = barWidth(bookTypeRows[i].Count, maxCount)
		}
	}
	data.BookTypeBreakdown = bookTypeRows

	// --- Collection Stats (scan dates as strings since SQLite returns text) ---
	var firstBookDate, lastReadDate sql.NullString
	if err := queryRowContext(ctx, s.db, "SELECT MIN(created_at) FROM books", &firstBookDate); err != nil {
		return nil, fmt.Errorf("first book date: %w", err)
	}

	if err := queryRowContext(ctx, s.db, "SELECT MAX(read_at) FROM reading_logs", &lastReadDate); err != nil {
		return nil, fmt.Errorf("last read date: %w", err)
	}

	var totalPages int
	if err := queryRowContext(ctx, s.db,
		"SELECT COALESCE(SUM(page_count), 0) FROM books WHERE page_count IS NOT NULL", &totalPages); err != nil {
		return nil, fmt.Errorf("total pages: %w", err)
	}

	var coversUploaded int
	if err := queryRowContext(ctx, s.db,
		"SELECT COUNT(*) FROM books WHERE cover_image_url IS NOT NULL", &coversUploaded); err != nil {
		return nil, fmt.Errorf("covers uploaded: %w", err)
	}

	var booksReadCount int
	if err := queryRowContext(ctx, s.db,
		"SELECT COUNT(DISTINCT book_id) FROM reading_logs", &booksReadCount); err != nil {
		return nil, fmt.Errorf("books read count: %w", err)
	}

	var booksRatedCount int
	if err := queryRowContext(ctx, s.db,
		"SELECT COUNT(*) FROM books WHERE child_rating IS NOT NULL AND child_rating >= 4", &booksRatedCount); err != nil {
		return nil, fmt.Errorf("books rated count: %w", err)
	}

	data.CollectionStats = []SectionRow{
		{Label: "First book added", Value: parseNullableDate(firstBookDate)},
		{Label: "Last read date", Value: parseNullableDate(lastReadDate)},
		{Label: "Total pages cataloged", Value: strconv.Itoa(totalPages) + " pages"},
		{Label: "Covers uploaded", Value: fmt.Sprintf("%d/%d", coversUploaded, bookCount)},
		{Label: "Books read ≥1 time", Value: fmt.Sprintf("%d/%d", booksReadCount, bookCount)},
		{Label: "Books rated ≥4★", Value: fmt.Sprintf("%d/%d", booksRatedCount, bookCount)},
	}

	// --- Books Read This Month ---
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	var booksReadThisMonth int
	if err := queryRowContext(ctx, s.db,
		"SELECT COUNT(DISTINCT book_id) FROM reading_logs WHERE read_at >= ? AND read_at < ?",
		&booksReadThisMonth, monthStart.Format("2006-01-02 15:04:05"), monthEnd.Format("2006-01-02 15:04:05")); err != nil {
		return nil, fmt.Errorf("books read this month: %w", err)
	}

	data.StatCards = append(data.StatCards, StatCard{
		Value: strconv.Itoa(booksReadThisMonth),
		Label: "Read This Month",
		Link:  "/reading-log",
	})

	return data, nil
}

// getGenreBreakdown returns genre counts. Returns empty slice on error instead of failing —
// some books may have malformed JSON in the genres column.
func (s *DashboardService) getGenreBreakdown(ctx context.Context) ([]GenreBreakdownRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT j.value AS genre, COUNT(*) AS cnt
		FROM books, json_each(books.genres) AS j
		GROUP BY j.value ORDER BY cnt DESC LIMIT 10
	`)
	if err != nil {
		return nil, fmt.Errorf("genre breakdown: %w", err)
	}

	var genreRows []GenreBreakdownRow
	for rows.Next() {
		var genre string
		var count int
		if err := rows.Scan(&genre, &count); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan genres: %w", err)
		}
		genreRows = append(genreRows, GenreBreakdownRow{Genre: genre, Count: count})
	}
	rows.Close()
	return genreRows, nil
}

// --- helpers ---

func queryRowContext(ctx context.Context, db *sql.DB, query string, dest interface{}, args ...interface{}) error {
	row := db.QueryRowContext(ctx, query, args...)
	return row.Scan(dest)
}

func pluralizeReads(count int) string {
	if count == 1 {
		return "1 read"
	}
	return fmt.Sprintf("%d reads", count)
}

func barWidth(value, max int) string {
	if max <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(value)/float64(max)*100)
}

func parseNullableDate(s sql.NullString) string {
	if !s.Valid || s.String == "" {
		return "—"
	}
	return parseDate(s.String)
}

func parseDate(s string) string {
	if s == "" {
		return "—"
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return s
	}
	return t.Format("Jan 2, 2006")
}
