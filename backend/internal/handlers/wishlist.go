package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.rcsmaine.com/chris/library/backend/internal/models"

	"github.com/go-chi/chi/v5"
)

const wishlistColumns = `
	id, title, author, isbn, reason, priority,
	amazon_url, thriftbooks_url, other_urls,
	cover_image_url, requested_by, requested_at,
	fulfilled, fulfilled_at, notes
`

// scanWishlistItem scans a row into a WishlistItem struct.
func scanWishlistItem(s scanner) (*models.WishlistItem, error) {
	var item models.WishlistItem
	var author sql.NullString
	var isbn sql.NullString
	var reason sql.NullString
	var amazonURL sql.NullString
	var thriftbooksURL sql.NullString
	var otherURLs sql.NullString
	var coverImageURL sql.NullString
	var requestedBy sql.NullString
	var fulfilledAt sql.NullTime
	var notes sql.NullString

	err := s.Scan(
		&item.ID,
		&item.Title,
		&author,
		&isbn,
		&reason,
		&item.Priority,
		&amazonURL,
		&thriftbooksURL,
		&otherURLs,
		&coverImageURL,
		&requestedBy,
		&item.RequestedAt,
		&item.Fulfilled,
		&fulfilledAt,
		&notes,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning wishlist item row: %w", err)
	}

	item.Author = nullStrPtr(author)
	item.ISBN = nullStrPtr(isbn)
	item.Reason = nullStrPtr(reason)
	item.AmazonURL = nullStrPtr(amazonURL)
	item.ThriftbooksURL = nullStrPtr(thriftbooksURL)
	item.OtherURLs = nullStrPtr(otherURLs)
	item.CoverImageURL = nullStrPtr(coverImageURL)
	item.RequestedBy = nullStrPtr(requestedBy)
	item.FulfilledAt = nullTimePtr(fulfilledAt)
	item.Notes = nullStrPtr(notes)

	return &item, nil
}

func nullTimePtr(nt sql.NullTime) *string {
	if nt.Valid {
		s := nt.Time.Format(time.RFC3339)
		return &s
	}
	return nil
}

// ListWishlistHandler returns all wishlist items ordered by priority DESC, then requested_at DESC.
func ListWishlistHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `SELECT ` + wishlistColumns + ` FROM wishlist ORDER BY priority DESC, requested_at DESC`

		rows, err := db.Query(query)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		defer rows.Close()

		items := make([]models.WishlistItem, 0)
		for rows.Next() {
			item, err := scanWishlistItem(rows)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			items = append(items, *item)
		}
		if err = rows.Err(); err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, items)
	}
}

// CreateWishlistItemHandler adds a new wishlist item.
func CreateWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateWishlistItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if strings.TrimSpace(req.Title) == "" {
			JSONError(w, http.StatusBadRequest, "title is required")
			return
		}

		query := `
			INSERT INTO wishlist (
				title, author, isbn, reason, priority,
				amazon_url, thriftbooks_url, other_urls, notes
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := db.Exec(query,
			req.Title,
			req.Author,
			req.ISBN,
			req.Reason,
			req.Priority,
			req.AmazonURL,
			req.ThriftbooksURL,
			req.OtherURLs,
			req.Notes,
		)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		// Return the created item
		row := db.QueryRow(`SELECT `+wishlistColumns+` FROM wishlist WHERE id = ?`, id)
		item, err := scanWishlistItem(row)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusCreated, item)
	}
}

// UpdateWishlistItemHandler updates an existing wishlist item.
func UpdateWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid wishlist item ID")
			return
		}

		var req models.UpdateWishlistItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Build dynamic UPDATE from non-nil fields.
		// Empty strings are treated as "set to NULL" so fields can be cleared.
		sets := []string{}
		args := []interface{}{}

		if req.Title != nil {
			sets = append(sets, "title = ?")
			args = append(args, ptrIfNonEmpty(*req.Title))
		}
		if req.Author != nil {
			sets = append(sets, "author = ?")
			args = append(args, ptrIfNonEmpty(*req.Author))
		}
		if req.ISBN != nil {
			sets = append(sets, "isbn = ?")
			args = append(args, ptrIfNonEmpty(*req.ISBN))
		}
		if req.Reason != nil {
			sets = append(sets, "reason = ?")
			args = append(args, ptrIfNonEmpty(*req.Reason))
		}
		if req.Priority != nil {
			sets = append(sets, "priority = ?")
			args = append(args, *req.Priority)
		}
		if req.AmazonURL != nil {
			sets = append(sets, "amazon_url = ?")
			args = append(args, ptrIfNonEmpty(*req.AmazonURL))
		}
		if req.ThriftbooksURL != nil {
			sets = append(sets, "thriftbooks_url = ?")
			args = append(args, ptrIfNonEmpty(*req.ThriftbooksURL))
		}
		if req.OtherURLs != nil {
			sets = append(sets, "other_urls = ?")
			args = append(args, ptrIfNonEmpty(*req.OtherURLs))
		}
		if req.CoverImageURL != nil {
			sets = append(sets, "cover_image_url = ?")
			args = append(args, ptrIfNonEmpty(*req.CoverImageURL))
		}
		if req.RequestedBy != nil {
			sets = append(sets, "requested_by = ?")
			args = append(args, ptrIfNonEmpty(*req.RequestedBy))
		}
		if req.Notes != nil {
			sets = append(sets, "notes = ?")
			args = append(args, ptrIfNonEmpty(*req.Notes))
		}

		if len(sets) == 0 {
			// No fields to update — just return the current item
			row := db.QueryRow(`SELECT `+wishlistColumns+` FROM wishlist WHERE id = ?`, id)
			item, err := scanWishlistItem(row)
			if err == sql.ErrNoRows {
				JSONError(w, http.StatusNotFound, "wishlist item not found")
				return
			}
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "database error")
				return
			}
			JSONResponse(w, http.StatusOK, item)
			return
		}

		args = append(args, id)
		// #nosec G202 -- Column names are validated against allowlist before use
		query := "UPDATE wishlist SET " + strings.Join(sets, ", ") + " WHERE id = ?"

		result, err := db.Exec(query, args...)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "wishlist item not found")
			return
		}

		// Return the updated item
		row := db.QueryRow(`SELECT `+wishlistColumns+` FROM wishlist WHERE id = ?`, id)
		item, err := scanWishlistItem(row)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, item)
	}
}

// DeleteWishlistItemHandler removes a wishlist item.
func DeleteWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid wishlist item ID")
			return
		}

		result, err := db.Exec("DELETE FROM wishlist WHERE id = ?", id)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "wishlist item not found")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "wishlist item deleted",
		})
	}
}

// FulfillWishlistItemHandler marks a wishlist item as fulfilled.
func FulfillWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid wishlist item ID")
			return
		}

		query := `UPDATE wishlist SET fulfilled = 1, fulfilled_at = CURRENT_TIMESTAMP WHERE id = ?`

		result, err := db.Exec(query, id)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "wishlist item not found")
			return
		}

		// Return the updated item
		row := db.QueryRow(`SELECT `+wishlistColumns+` FROM wishlist WHERE id = ?`, id)
		item, err := scanWishlistItem(row)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, item)
	}
}

// HTMLWishlistFormHandler returns a modal HTML fragment for creating a wishlist item.
// Optionally pre-fills fields from a book_id query parameter.
func HTMLWishlistFormHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookID := r.URL.Query().Get("book_id")

		var title, author, isbn, coverURL string
		if bookID != "" {
			row := db.QueryRow(`SELECT title, authors, isbn, cover_image_url FROM books WHERE id = ?`, bookID)
			var authors sql.NullString
			var nullISBN, nullCover sql.NullString
			if err := row.Scan(&title, &authors, &nullISBN, &nullCover); err == nil {
				if authors.Valid {
					author = authors.String
				}
				if nullISBN.Valid {
					isbn = nullISBN.String
				}
				if nullCover.Valid {
					coverURL = nullCover.String
				}
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
<div id="modal-backdrop" class="modal-backdrop" hx-on::click="if(event.target===this)document.getElementById('modal-backdrop').remove()">
  <div class="modal-content modal-sm p-6" role="dialog" aria-modal="true">
    <div class="flex items-center justify-between mb-4 pb-3 border-b" style="border-color: rgba(139, 69, 19, 0.1);">
      <h2 class="text-xl font-heading font-semibold text-primary">Add to Wishlist</h2>
      <button type="button" hx-on::click="document.getElementById('modal-backdrop').remove()" class="text-text-light hover:text-text transition-colors text-2xl no-underline" aria-label="Close modal">×</button>
    </div>
    <form hx-post="/wishlist/create" hx-target="#modal-target" hx-swap="outerHTML">
      <div class="space-y-4">
        ` + (func() string {
			if coverURL != "" {
				return `<div class="flex justify-center mb-2">
              <img src="` + html.EscapeString(coverURL) + `" alt="Cover preview" class="w-24 h-36 object-cover rounded-lg shadow-sm border" style="border-color: rgba(139, 69, 19, 0.1);">
            </div>`
			}
			return ""
		})() + `
        <div>
          <label for="wl-title" class="block text-sm font-medium text-text mb-1">Title <span class="text-error">*</span></label>
          <input type="text" id="wl-title" name="title" value="` + html.EscapeString(title) + `" required class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="wl-author" class="block text-sm font-medium text-text mb-1">Author</label>
          <input type="text" id="wl-author" name="author" value="` + html.EscapeString(author) + `" class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="wl-isbn" class="block text-sm font-medium text-text mb-1">ISBN</label>
          <input type="text" id="wl-isbn" name="isbn" value="` + html.EscapeString(isbn) + `" class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="wl-reason" class="block text-sm font-medium text-text mb-1">Reason</label>
          <input type="text" id="wl-reason" name="reason" placeholder='Why do you want this book?' class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="wl-priority" class="block text-sm font-medium text-text mb-1">Priority (1-5)</label>
          <input type="number" id="wl-priority" name="priority" value="3" min="1" max="5" class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="wl-notes" class="block text-sm font-medium text-text mb-1">Notes</label>
          <textarea id="wl-notes" name="notes" rows="2" class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);"></textarea>
        </div>
      </div>
      <div id="wishlist-form-error" class="mt-4"></div>
      <div class="flex justify-end gap-3 mt-6">
        <button type="button" hx-on::click="document.getElementById('modal-backdrop').remove()" class="px-4 py-2 rounded-lg border text-text-light hover:text-text transition-colors no-underline" style="border-color: var(--color-secondary);">Cancel</button>
        <button type="submit" class="px-4 py-2 rounded-lg font-medium text-white" style="background-color: var(--color-primary);">Add to Wishlist</button>
      </div>
    </form>
  </div>
</div>`))
	}
}

// HTMLCreateWishlistItemHandler creates a wishlist item from form-encoded data.
// On success: returns a success toast and redirects the wishlist list.
// On failure: returns an HTML error fragment.
func HTMLCreateWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Title is required")))
			return
		}

		author := ptrIfNonEmpty(r.FormValue("author"))
		isbn := ptrIfNonEmpty(r.FormValue("isbn"))
		reason := ptrIfNonEmpty(r.FormValue("reason"))
		notes := ptrIfNonEmpty(r.FormValue("notes"))

		priority := 3
		if p := strings.TrimSpace(r.FormValue("priority")); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				if parsed >= 1 && parsed <= 5 {
					priority = parsed
				}
			}
		}

		query := `
			INSERT INTO wishlist (title, author, isbn, reason, priority, notes)
			VALUES (?, ?, ?, ?, ?, ?)
		`
		_, err := db.Exec(query, title, author, isbn, reason, priority, notes)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to add to wishlist")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(htmlSuccessToast("Added to wishlist!")))
	}
}
