package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func nullTimePtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
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

		// Build dynamic UPDATE from non-nil fields
		sets := []string{}
		args := []interface{}{}

		if req.Title != nil {
			sets = append(sets, "title = ?")
			args = append(args, *req.Title)
		}
		if req.Author != nil {
			sets = append(sets, "author = ?")
			args = append(args, *req.Author)
		}
		if req.ISBN != nil {
			sets = append(sets, "isbn = ?")
			args = append(args, *req.ISBN)
		}
		if req.Reason != nil {
			sets = append(sets, "reason = ?")
			args = append(args, *req.Reason)
		}
		if req.Priority != nil {
			sets = append(sets, "priority = ?")
			args = append(args, *req.Priority)
		}
		if req.AmazonURL != nil {
			sets = append(sets, "amazon_url = ?")
			args = append(args, *req.AmazonURL)
		}
		if req.ThriftbooksURL != nil {
			sets = append(sets, "thriftbooks_url = ?")
			args = append(args, *req.ThriftbooksURL)
		}
		if req.OtherURLs != nil {
			sets = append(sets, "other_urls = ?")
			args = append(args, *req.OtherURLs)
		}
		if req.CoverImageURL != nil {
			sets = append(sets, "cover_image_url = ?")
			args = append(args, *req.CoverImageURL)
		}
		if req.RequestedBy != nil {
			sets = append(sets, "requested_by = ?")
			args = append(args, *req.RequestedBy)
		}
		if req.Notes != nil {
			sets = append(sets, "notes = ?")
			args = append(args, *req.Notes)
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
