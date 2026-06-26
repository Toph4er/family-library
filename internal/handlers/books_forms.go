package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// HTMLCreateBookHandler handles POST /books/create.
func HTMLCreateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			HTMXErrorResponse(w, http.StatusBadRequest, "Invalid request")
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			HTMXErrorResponse(w, http.StatusBadRequest, "Title is required")
			return
		}

		isbn := strings.ReplaceAll(strings.TrimSpace(r.FormValue("isbn")), "-", "")
		if isbn == "" {
			HTMXErrorResponse(w, http.StatusBadRequest, "ISBN is required")
			return
		}

		coverSource := "none"
		childRatingStr := r.FormValue("child_rating")
		var childRating *int
		if childRatingStr != "" {
			if v, err := strconv.Atoi(childRatingStr); err == nil {
				childRating = &v
			}
		}
		pubYearStr := r.FormValue("publication_year")
		var pubYear *int
		if pubYearStr != "" {
			if v, err := strconv.Atoi(pubYearStr); err == nil {
				pubYear = &v
			}
		}
		pageCountStr := r.FormValue("page_count")
		var pageCount *int
		if pageCountStr != "" {
			if v, err := strconv.Atoi(pageCountStr); err == nil {
				pageCount = &v
			}
		}

		query := `
			INSERT INTO books (
				isbn, title, subtitle, authors, illustrators,
				publisher, publication_year, page_count, book_type,
				reading_levels, genres, themes, awards,
				gift_from, gift_relationship, date_received,
				condition, location, notes,
				child_rating, quantity, read_count, last_read_date, cover_image_url, cover_source, dewey_decimal_class, language,
				subject_places, subject_people, subject_times, series, age_range, guest_visible_fields
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		result, err := db.Exec(query,
			ptrIfNonEmpty(isbn), title,
			ptrIfNonEmpty(r.FormValue("subtitle")),
			ptrIfNonEmpty(r.FormValue("authors")),
			ptrIfNonEmpty(r.FormValue("illustrators")),
			ptrIfNonEmpty(r.FormValue("publisher")),
			pubYear, pageCount,
			ptrIfNonEmpty(r.FormValue("book_type")),
			ptrIfNonEmpty(r.FormValue("reading_levels")),
			ptrIfNonEmpty(r.FormValue("genres")),
			ptrIfNonEmpty(r.FormValue("themes")),
			ptrIfNonEmpty(r.FormValue("awards")),
			ptrIfNonEmpty(r.FormValue("gift_from")),
			ptrIfNonEmpty(r.FormValue("gift_relationship")),
			ptrIfNonEmpty(r.FormValue("date_received")),
			ptrIfNonEmpty(r.FormValue("condition")),
			ptrIfNonEmpty(r.FormValue("location")),
			ptrIfNonEmpty(r.FormValue("notes")),
			childRating,
			nil, // last_read_date
			ptrIfNonEmpty(r.FormValue("cover_image_url")),
			coverSource,
			ptrIfNonEmpty(r.FormValue("dewey_decimal_class")),
			ptrIfNonEmpty(r.FormValue("language")),
			ptrIfNonEmpty(r.FormValue("subject_places")),
			ptrIfNonEmpty(r.FormValue("subject_people")),
			ptrIfNonEmpty(r.FormValue("subject_times")),
			ptrIfNonEmpty(r.FormValue("series")),
			ptrIfNonEmpty(r.FormValue("age_range")),
			defaultGuestVisibleFields(),
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				HTMXErrorResponse(w, http.StatusConflict, "A book with this ISBN already exists")
				return
			}
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to create book")
		}

		id, _ := result.LastInsertId()
		w.Header().Set("HX-Redirect", "/books/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLUpdateBookHandler handles POST /books/{id}/update.
func HTMLUpdateBookHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			HTMXErrorResponse(w, http.StatusBadRequest, "Invalid book ID")
			return
		}

		if err := r.ParseForm(); err != nil {
			HTMXErrorResponse(w, http.StatusBadRequest, "Invalid request")
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			HTMXErrorResponse(w, http.StatusBadRequest, "Title is required")
			return
		}
		isbn := strings.ReplaceAll(strings.TrimSpace(r.FormValue("isbn")), "-", "")
		if isbn == "" {
			HTMXErrorResponse(w, http.StatusBadRequest, "ISBN is required")
			return
		}

		sets := []updateField{
			{column: "isbn", value: ptrIfNonEmpty(isbn)},
			{column: "title", value: ptrIfNonEmpty(title)},
		}

		stringFields := []string{
			"subtitle", "authors", "illustrators", "publisher", "book_type",
			"reading_levels", "genres", "themes", "awards", "gift_from",
			"gift_relationship", "date_received", "condition", "location",
			"notes", "cover_image_url", "dewey_decimal_class", "language",
			"subject_places", "subject_people", "subject_times", "series", "age_range",
		}
		for _, name := range stringFields {
			sets = append(sets, updateField{column: name, value: ptrIfNonEmpty(strings.TrimSpace(r.FormValue(name)))})
		}

		for _, name := range []string{"publication_year", "page_count", "child_rating"} {
			v := strings.TrimSpace(r.FormValue(name))
			if v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					sets = append(sets, updateField{column: name, value: &n})
				} else {
					sets = append(sets, updateField{column: name, value: nil})
				}
			} else {
				sets = append(sets, updateField{column: name, value: nil})
			}
		}

		qv := strings.TrimSpace(r.FormValue("quantity"))
		if qv != "" {
			if n, err := strconv.Atoi(qv); err == nil && n >= 1 {
				sets = append(sets, updateField{column: "quantity", value: &n})
			} else {
				sets = append(sets, updateField{column: "quantity", value: nil})
			}
		} else {
			sets = append(sets, updateField{column: "quantity", value: nil})
		}

		sets = append(sets, rawField("updated_at", "CURRENT_TIMESTAMP"))

		setClause, args := buildUpdateClauses(sets)
		args = append(args, id)

		query := "UPDATE books SET " + setClause + " WHERE id = ?" // #nosec G202 -- Column names are hardcoded
		result, err := db.Exec(query, args...)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				HTMXErrorResponse(w, http.StatusConflict, "A book with this ISBN already exists")
				return
			}
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to update book")
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			HTMXErrorResponse(w, http.StatusNotFound, "Book not found")
			return
		}

		w.Header().Set("HX-Redirect", "/books/"+strconv.FormatInt(id, 10))
		w.WriteHeader(http.StatusOK)
	}
}
