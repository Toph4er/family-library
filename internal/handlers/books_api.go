package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Toph4er/family-library/internal/repository"
	"github.com/Toph4er/family-library/internal/validation"
)

// defaultGuestVisibleFields returns the default JSON blob for guest visibility.
func defaultGuestVisibleFields() string {
	fields := map[string]bool{
		"title":               true,
		"subtitle":            true,
		"authors":             true,
		"illustrators":        true,
		"publisher":           true,
		"publication_year":    true,
		"page_count":          true,
		"quantity":            true,
		"book_type":           true,
		"reading_levels":      true,
		"genres":              true,
		"themes":              true,
		"awards":              true,
		"gift_from":           true,
		"gift_relationship":   true,
		"child_rating":        true,
		"read_count":          true,
		"cover_image_url":     true,
		"cover_source":        true,
		"dewey_decimal_class": true,
		"description":         true,
		"language":            true,
		"subject_places":      true,
		"subject_people":      true,
		"subject_times":       true,
		"series":              true,
		"age_range":           true,
		"isbn":                false,
		"condition":           false,
		"location":            false,
		"notes":               false,
		"date_received":       false,
		"last_read_date":      false,
	}
	b, _ := json.Marshal(fields)
	return string(b)
}

// LookupISBNHandler looks up book metadata by ISBN without creating a record.
func LookupISBNHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isbn := strings.ReplaceAll(strings.TrimSpace(r.URL.Query().Get("isbn")), "-", "")

		var errs validation.Errors
		errs.Required("isbn", isbn)
		if errs.HasErrors() {
			JSONError(w, http.StatusBadRequest, errs.First())
			return
		}
		force := r.URL.Query().Get("force") == "true"

		book, coverSource, apiErr := cachedFetchFromOpenLibrary(db, isbn, force)
		if apiErr != nil {
			slog.Error("Open Library lookup failed", "isbn", isbn, "error", apiErr)
			JSONError(w, http.StatusBadGateway, fmt.Sprintf("book lookup unavailable: %v", apiErr))
			return
		}
		if book == nil {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}

		resp := buildLookupResponse(book, coverSource)
		JSONResponse(w, http.StatusOK, resp)
	}
}

// DeleteBookHandler deletes a book by ID.
func DeleteBookHandler(repo repository.BookRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			HTMXErrorResponse(w, http.StatusBadRequest, "Invalid book ID")
			return
		}

		err = repo.Delete(r.Context(), id)
		if err != nil {
			if strings.Contains(err.Error(), "no rows affected") {
				HTMXErrorResponse(w, http.StatusNotFound, "Book not found")
				return
			}
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to delete book")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Redirect", "/books")
		w.WriteHeader(http.StatusOK)
	}
}

// RateChildHandler updates the child_rating for a book.
func RateChildHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var bookID int64
		var rating int

		if r.Header.Get("Content-Type") == "application/json" {
			var req struct {
				BookID int64 `json:"book_id"`
				Rating int   `json:"rating"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				JSONError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			bookID = req.BookID
			rating = req.Rating
		} else {
			var err error
			bookID, err = strconv.ParseInt(r.FormValue("book_id"), 10, 64)
			if err != nil || bookID <= 0 {
				JSONError(w, http.StatusBadRequest, "book_id is required")
				return
			}
			rating, err = strconv.Atoi(r.FormValue("rating"))
			if err != nil || rating < 1 || rating > 5 {
				JSONError(w, http.StatusBadRequest, "rating must be between 1 and 5")
				return
			}
		}

		var errs validation.Errors
		errs.Positive("book_id", bookID)
		if rating != 0 {
			errs.InRange("rating", rating, 1, 5)
		} else {
			errs.Required("rating", "")
		}
		if errs.HasErrors() {
			JSONError(w, http.StatusBadRequest, errs.First())
			return
		}

		result, err := db.Exec("UPDATE books SET child_rating = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", rating, bookID)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "book not found")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "rating updated",
		})
	}
}
