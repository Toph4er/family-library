package pages

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gorilla/sessions"

	sqldb "github.com/Toph4er/family-library/internal/db"
	"github.com/Toph4er/family-library/internal/models"
)

// RenderWishlistPage renders the wishlist page (open to guests).
func RenderWishlistPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		rows, err := db.Query("SELECT id, isbn, title, author, reason, priority, amazon_url, thriftbooks_url, notes, fulfilled, requested_by, requested_at, fulfilled_at, cover_image_url FROM wishlist ORDER BY priority DESC, requested_at DESC")
		if err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
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
				renderHTMXError(w, r, http.StatusInternalServerError)
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
			renderHTMXError(w, r, http.StatusInternalServerError)
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
			renderHTMXError(w, r, http.StatusInternalServerError)
		}
	}
}
