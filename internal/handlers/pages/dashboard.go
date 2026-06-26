package pages

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/Toph4er/family-library/internal/services"
)

// RenderDashboardPage renders the dashboard page (auth required).
func RenderDashboardPage(tmpl *template.Template, dashSvc *services.DashboardService, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, nil) // theme loaded via middleware path for protected routes
		ctx := pageContext{BaseContext: base}

		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		data, err := dashSvc.Get(r.Context())
		if err != nil {
			slog.Error("dashboard service error", "error", err)
			renderHTMXError(w, r, 500)
			return
		}

		dash := DashboardContext{}

		// Map service StatCards → page StatCards (add icons based on label)
		for _, sc := range data.StatCards {
			dash.StatCards = append(dash.StatCards, StatCard{
				Icon:  iconForLabel(sc.Label),
				Value: sc.Value,
				Label: sc.Label,
				Link:  sc.Link,
			})
		}

		// Map section rows with icons and titles
		dash.Sections = []SectionCard{
			{Icon: svgIcon("open-book"), Title: "Most Read Books", Rows: toPageRows(data.MostRead), Link: "/reading-log"},
			{Icon: svgIcon("star"), Title: "Highest Rated", Rows: toPageRows(data.HighestRated), Link: "/books"},
			{Icon: svgIcon("sparkles"), Title: "Recently Added", Rows: toPageRows(data.RecentlyAdded), Link: "/books"},
			{Icon: svgIcon("clipboard-list"), Title: "Most Wanted", Rows: toPageRows(data.MostWanted), Link: "/wishlist"},
		}

		dash.CollectionStats = toPageRows(data.CollectionStats)

		for _, rb := range data.ReaderBreakdown {
			dash.ReaderBreakdown = append(dash.ReaderBreakdown, ReaderBreakdownRow{
				Reader: rb.Reader, Count: rb.Count, Width: rb.Width,
			})
		}
		for _, gb := range data.GenreBreakdown {
			dash.GenreBreakdown = append(dash.GenreBreakdown, GenreBreakdownRow{
				Genre: gb.Genre, Count: gb.Count, Width: gb.Width,
			})
		}
		for _, bt := range data.BookTypeBreakdown {
			dash.BookTypeBreakdown = append(dash.BookTypeBreakdown, BookTypeBreakdownRow{
				BookType: bt.BookType, Count: bt.Count, Width: bt.Width,
			})
		}

		ctx.DashboardContext = dash
		renderPage(w, r, tmpl, "dashboard.html", ctx)
	}
}

func iconForLabel(label string) template.HTML {
	switch label {
	case "Total Books":
		return svgIcon("book")
	case "Total Reads":
		return svgIcon("open-book")
	case "Avg Rating":
		return svgIcon("star")
	case "Wishlist Items":
		return svgIcon("clipboard-list")
	case "Read This Month":
		return svgIcon("calendar")
	default:
		return ""
	}
}

func toPageRows(rows []services.SectionRow) []SectionRow {
	result := make([]SectionRow, len(rows))
	for i, r := range rows {
		result[i] = SectionRow{Label: r.Label, Value: r.Value, Link: r.Link}
	}
	return result
}
