package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"git.rcsmaine.com/chris/library/internal/db"
	"git.rcsmaine.com/chris/library/internal/models"

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

	item.Author = db.NullStrPtr(author)
	item.ISBN = db.NullStrPtr(isbn)
	item.Reason = db.NullStrPtr(reason)
	item.AmazonURL = db.NullStrPtr(amazonURL)
	item.ThriftbooksURL = db.NullStrPtr(thriftbooksURL)
	item.OtherURLs = db.NullStrPtr(otherURLs)
	item.CoverImageURL = db.NullStrPtr(coverImageURL)
	item.RequestedBy = db.NullStrPtr(requestedBy)
	item.FulfilledAt = db.NullTimePtr(fulfilledAt)
	item.Notes = db.NullStrPtr(notes)

	return &item, nil
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
		amazonURL := ptrIfNonEmpty(r.FormValue("amazon_url"))
		thriftbooksURL := ptrIfNonEmpty(r.FormValue("thriftbooks_url"))
		coverImageURL := ptrIfNonEmpty(r.FormValue("cover_image_url"))

		priority := 3
		if p := strings.TrimSpace(r.FormValue("priority")); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				if parsed >= 1 && parsed <= 5 {
					priority = parsed
				}
			}
		}

		query := `
			INSERT INTO wishlist (title, author, isbn, reason, priority,
				amazon_url, thriftbooks_url, cover_image_url, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err := db.Exec(query, title, author, isbn, reason, priority,
			amazonURL, thriftbooksURL, coverImageURL, notes)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to add to wishlist")))
			return
		}

		w.Header().Set("HX-Redirect", "/wishlist")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLUpdateWishlistItemHandler updates a wishlist item from form-encoded data.
// On success: redirects to the wishlist page.
// On failure: returns an HTML error fragment.
func HTMLUpdateWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("Wishlist item not found")))
			return
		}

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
		amazonURL := ptrIfNonEmpty(r.FormValue("amazon_url"))
		thriftbooksURL := ptrIfNonEmpty(r.FormValue("thriftbooks_url"))
		coverImageURL := ptrIfNonEmpty(r.FormValue("cover_image_url"))

		priority := 3
		if p := strings.TrimSpace(r.FormValue("priority")); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				if parsed >= 1 && parsed <= 5 {
					priority = parsed
				}
			}
		}

		query := `
			UPDATE wishlist SET
				title = ?, author = ?, isbn = ?, reason = ?, priority = ?,
				amazon_url = ?, thriftbooks_url = ?, cover_image_url = ?, notes = ?
			WHERE id = ?
		`
		result, err := db.Exec(query, title, author, isbn, reason, priority,
			amazonURL, thriftbooksURL, coverImageURL, notes, id)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to update wishlist item")))
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("Wishlist item not found")))
			return
		}

		w.Header().Set("HX-Redirect", "/wishlist")
		w.WriteHeader(http.StatusOK)
	}
}
