package pages

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
)

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
