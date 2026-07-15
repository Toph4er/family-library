package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	pages "github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/models"
	"github.com/Toph4er/family-library/internal/validation"
)

const readingLogColumns = `rl.id, rl.book_id, rl.start_page, rl.end_page, rl.total_pages, rl.entire_book, rl.read_at, rl.reader_name, rl.notes, rl.created_at`

// --- JSON API Handlers ---

// ListReadingLogsHandler returns all reading log entries.
func ListReadingLogsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `SELECT ` + readingLogColumns + `, b.title FROM reading_logs rl JOIN books b ON b.id = rl.book_id ORDER BY rl.read_at DESC`
		rows, err := db.Query(query)
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}
		defer rows.Close()

		logs := make([]models.ReadingLog, 0)
		for rows.Next() {
			var rl models.ReadingLog
			var entireBook int
			err := rows.Scan(&rl.ID, &rl.BookID, &rl.StartPage, &rl.EndPage, &rl.TotalPages, &entireBook, &rl.ReadAt, &rl.ReaderName, &rl.Notes, &rl.CreatedAt, &rl.BookTitle)
			if err != nil {
				JSONError(w, r, http.StatusInternalServerError, "database error")
				return
			}
			rl.EntireBook = entireBook != 0
			logs = append(logs, rl)
		}
		if err = rows.Err(); err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, logs)
	}
}

// CreateReadingLogHandler creates a new reading log entry.
func CreateReadingLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateReadingLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.BookID == 0 {
			JSONError(w, r, http.StatusBadRequest, "book_id is required")
			return
		}

		var errs validation.Errors
		errs.Required("reader_name", strings.TrimSpace(req.ReaderName))
		if errs.HasErrors() {
			JSONError(w, r, http.StatusBadRequest, errs.First())
			return
		}

		// Validate book exists
		var bookTitle string
		err := db.QueryRow("SELECT title FROM books WHERE id = ?", req.BookID).Scan(&bookTitle)
		if errors.Is(err, sql.ErrNoRows) {
			JSONError(w, r, http.StatusNotFound, "book not found")
			return
		}
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		// Calculate total pages
		totalPages := 0
		if req.StartPage != nil && req.EndPage != nil {
			totalPages = *req.EndPage - *req.StartPage + 1
			if totalPages < 0 {
				totalPages = 0
			}
		}

		entireBook := 0
		if req.EntireBook {
			entireBook = 1
		}

		readAt := req.ReadAt
		if readAt == "" {
			readAt = time.Now().Format("2006-01-02 15:04:05")
		}

		result, err := db.Exec(
			"INSERT INTO reading_logs (book_id, start_page, end_page, total_pages, entire_book, read_at, reader_name, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			req.BookID, req.StartPage, req.EndPage, totalPages, entireBook, readAt, strings.TrimSpace(req.ReaderName), req.Notes,
		)
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		row := db.QueryRow("SELECT "+readingLogColumns+", b.title FROM reading_logs rl JOIN books b ON b.id = rl.book_id WHERE rl.id = ?", id)
		rl, err := func() (*models.ReadingLog, error) {
			var r models.ReadingLog
			var eb int
			err := row.Scan(&r.ID, &r.BookID, &r.StartPage, &r.EndPage, &r.TotalPages, &eb, &r.ReadAt, &r.ReaderName, &r.Notes, &r.CreatedAt, &r.BookTitle)
			r.EntireBook = eb != 0
			return &r, err
		}()
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusCreated, rl)
	}
}

// UpdateReadingLogHandler updates a reading log entry.
func UpdateReadingLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, r, http.StatusBadRequest, "invalid ID")
			return
		}

		var req models.UpdateReadingLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, r, http.StatusBadRequest, "invalid request body")
			return
		}

		var fields []updateField

		if req.StartPage != nil || req.EndPage != nil {
			// Recalculate total_pages when either changes
			var sp, ep *int
			if req.StartPage != nil {
				sp = req.StartPage
			} else {
				var existing int
				err := db.QueryRow("SELECT start_page FROM reading_logs WHERE id = ?", id).Scan(&existing)
				if err == nil {
					sp = &existing
				}
			}
			if req.EndPage != nil {
				ep = req.EndPage
			} else {
				var existing int
				err := db.QueryRow("SELECT end_page FROM reading_logs WHERE id = ?", id).Scan(&existing)
				if err == nil {
					ep = &existing
				}
			}

			if req.StartPage != nil {
				fields = append(fields, updateField{column: "start_page", value: *req.StartPage})
			}
			if req.EndPage != nil {
				fields = append(fields, updateField{column: "end_page", value: *req.EndPage})
			}

			// Calculate new total
			if sp != nil && ep != nil {
				tot := *ep - *sp + 1
				if tot < 0 {
					tot = 0
				}
				fields = append(fields, updateField{column: "total_pages", value: tot})
			}
		}

		if req.EntireBook != nil {
			val := 0
			if *req.EntireBook {
				val = 1
			}
			fields = append(fields, updateField{column: "entire_book", value: val})
		}
		if req.ReadAt != nil {
			fields = append(fields, updateField{column: "read_at", value: *req.ReadAt})
		}
		if req.ReaderName != nil {
			val := strings.TrimSpace(*req.ReaderName)
			var errs validation.Errors
			errs.Required("reader_name", val)
			if errs.HasErrors() {
				JSONError(w, r, http.StatusBadRequest, errs.First())
				return
			}
			fields = append(fields, updateField{column: "reader_name", value: val})
		}
		if req.Notes != nil {
			fields = append(fields, updateField{column: "notes", value: *req.Notes})
		}

		if len(fields) == 0 {
			row := db.QueryRow("SELECT "+readingLogColumns+", b.title FROM reading_logs rl JOIN books b ON b.id = rl.book_id WHERE rl.id = ?", id)
			rl, err := func() (*models.ReadingLog, error) {
				var r models.ReadingLog
				var eb int
				err := row.Scan(&r.ID, &r.BookID, &r.StartPage, &r.EndPage, &r.TotalPages, &eb, &r.ReadAt, &r.ReaderName, &r.Notes, &r.CreatedAt, &r.BookTitle)
				r.EntireBook = eb != 0
				return &r, err
			}()
			if errors.Is(err, sql.ErrNoRows) {
				JSONError(w, r, http.StatusNotFound, "reading log not found")
				return
			}
			if err != nil {
				JSONError(w, r, http.StatusInternalServerError, "database error")
				return
			}
			JSONResponse(w, http.StatusOK, rl)
			return
		}

		setClause, args := buildUpdateClauses(fields)
		args = append(args, id)
		// #nosec G202 -- Column names are hardcoded constants, not user input
		query := "UPDATE reading_logs SET " + setClause + " WHERE id = ?"
		result, err := db.Exec(query, args...)
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			JSONError(w, r, http.StatusNotFound, "reading log not found")
			return
		}

		row := db.QueryRow("SELECT "+readingLogColumns+", b.title FROM reading_logs rl JOIN books b ON b.id = rl.book_id WHERE rl.id = ?", id)
		rl, err := func() (*models.ReadingLog, error) {
			var r models.ReadingLog
			var eb int
			err := row.Scan(&r.ID, &r.BookID, &r.StartPage, &r.EndPage, &r.TotalPages, &eb, &r.ReadAt, &r.ReaderName, &r.Notes, &r.CreatedAt, &r.BookTitle)
			r.EntireBook = eb != 0
			return &r, err
		}()
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		JSONResponse(w, http.StatusOK, rl)
	}
}

// DeleteReadingLogHandler deletes a reading log entry.
func DeleteReadingLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			JSONError(w, r, http.StatusBadRequest, "invalid ID")
			return
		}

		result, err := db.Exec("DELETE FROM reading_logs WHERE id = ?", id)
		if err != nil {
			JSONError(w, r, http.StatusInternalServerError, "database error")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			JSONError(w, r, http.StatusNotFound, "reading log not found")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "reading log deleted",
		})
	}
}

// --- HTMX HTML Handlers ---

// HTMLBookSelectorHandler returns a modal HTML fragment listing recent books for logging a reading session.
// GET /reading-logs/book-selector
func HTMLBookSelectorHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, title FROM books ORDER BY title ASC LIMIT 50")
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var bookOptions string
		hasBooks := false
		for rows.Next() {
			var id int64
			var title string
			if err := rows.Scan(&id, &title); err != nil {
				continue
			}
			hasBooks = true
			bookOptions += fmt.Sprintf(`<a href="#" class="block px-4 py-3 rounded-lg hover:bg-background/50 transition-colors text-sm text-text no-underline" hx-on:click="openLogForm(%d); this.closest('.modal-backdrop').remove(); return false;">%s</a>`, id, template.HTMLEscapeString(title))
		}

		if !hasBooks {
			bookOptions = `<p class="text-text-light/60 text-sm text-center py-4">No books in the library yet.</p>`
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
<div id="modal-backdrop" class="modal-backdrop" hx-on:click="if(event.target===this)this.closest('.modal-backdrop').remove()">
  <div class="modal-content modal-md p-6" role="dialog" aria-modal="true">
    <div class="flex items-center justify-between mb-4 pb-3 border-b" style="border-color: var(--color-secondary);">
      <h2 class="text-xl font-heading font-semibold text-primary">Select a Book</h2>
      <button type="button" hx-on:click="this.closest('.modal-backdrop').remove()" class="text-text-light hover:text-text transition-colors text-2xl no-underline" aria-label="Close modal">×</button>
    </div>
    <div class="max-h-96 overflow-y-auto space-y-2">
      ` + bookOptions +
			`</div>
  </div>
</div>`))
	}
}

// HTMLReadingLogFormHandler returns a modal HTML fragment for adding/editing a reading log entry.
func HTMLReadingLogFormHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookIDStr := chi.URLParam(r, "book_id")
		if bookIDStr == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Book ID is required")))
			return
		}

		bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid book ID")))
			return
		}

		// Load book title
		var bookTitle string
		var pageCount *int
		err = db.QueryRow("SELECT title, page_count FROM books WHERE id = ?", bookID).Scan(&bookTitle, &pageCount)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}

		// Load family members for reader dropdown
		fmRows, err := db.Query("SELECT id, name, relation FROM family_members ORDER BY name ASC")
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer fmRows.Close()

		var fmOptions string
		for fmRows.Next() {
			var id int64
			var name, relation string
			if err := fmRows.Scan(&id, &name, &relation); err == nil {
				fmOptions += fmt.Sprintf(`<option value="%s">%s (%s)</option>`, template.HTMLEscapeString(name), template.HTMLEscapeString(name), template.HTMLEscapeString(relation))
			}
		}

		// Load user timezone for the datetime-local default
		var userTZ string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "user_timezone").Scan(&userTZ); err != nil {
			userTZ = "America/New_York"
		}
		if userTZ == "" {
			userTZ = "America/New_York"
		}
		tz, tzErr := time.LoadLocation(userTZ)
		if tzErr != nil {
			tz = time.Local
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		maxPage := ""
		endPagePlaceholder := ""
		if pageCount != nil && *pageCount > 0 {
			maxPage = fmt.Sprintf(" max=\"%d\"", *pageCount)
			endPagePlaceholder = strconv.Itoa(*pageCount)
		}

		// #nosec G705 -- All interpolated values are escaped via template.HTMLEscapeString()
		_, _ = w.Write([]byte(`
<div id="modal-backdrop" class="modal-backdrop" hx-on:click="if(event.target===this)document.getElementById('modal-backdrop').remove()">
  <div class="modal-content modal-md p-6" role="dialog" aria-modal="true">
    <div class="flex items-center justify-between mb-4 pb-3 border-b" style="border-color: rgba(139, 69, 19, 0.1);">
      <h2 class="text-xl font-heading font-semibold text-primary">Log Reading</h2>
      <button type="button" hx-on:click="document.getElementById('modal-backdrop').remove()" class="text-text-light hover:text-text transition-colors text-2xl no-underline" aria-label="Close modal">×</button>
    </div>
    <p class="text-sm text-text-light mb-4">Book: <strong class="text-text">` + template.HTMLEscapeString(bookTitle) + `</strong></p>
    <form hx-post="/reading-logs" hx-target="#modal-target" hx-swap="outerHTML">
      <input type="hidden" name="book_id" value="` + bookIDStr + `">
      <div class="space-y-4">
        <div id="page-fields" class="grid grid-cols-2 gap-3">
          <div>
            <label for="rl-start-page" class="block text-sm font-medium text-text mb-1">Start Page</label>
            <input type="number" id="rl-start-page" name="start_page" min="1"` + maxPage + ` class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" placeholder="1">
          </div>
          <div>
            <label for="rl-end-page" class="block text-sm font-medium text-text mb-1">End Page</label>
            <input type="number" id="rl-end-page" name="end_page" min="1"` + maxPage + ` class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" placeholder="` + endPagePlaceholder + `">
          </div>
        </div>
        <div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" name="entire_book" value="true" class="w-4 h-4 rounded border-secondary text-primary focus:ring-primary/20" hx-on:change="var f=document.getElementById('page-fields'); if(f)f.style.display=this.checked?'none':''">
            <span class="text-sm text-text">Read entire book</span>
          </label>
        </div>
        <div>
          <label for="rl-read-at" class="block text-sm font-medium text-text mb-1">Date &amp; Time</label>
          <input type="datetime-local" id="rl-read-at" name="read_at" value="` + time.Now().In(tz).Format("2006-01-02T15:04") + `" class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="rl-reader" class="block text-sm font-medium text-text mb-1">Reader <span class="text-error">*</span></label>
          <select id="rl-reader" name="reader_name" required class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" hx-on:change="var o=document.getElementById('rl-reader-other'); if(o)o.classList.toggle('hidden',this.value!=='other'); if(this.value==='other'&&o)o.focus()">
            <option value="">Select reader...</option>` + fmOptions + `
            <option value="other">Other (type manually)</option>
          </select>
          <input type="text" id="rl-reader-other" name="reader_name_manual" class="w-full px-3 py-2 rounded-lg border bg-surface text-sm mt-2 hidden" style="border-color: var(--color-secondary);" placeholder="Enter reader name">
        </div>
        <div>
          <label for="rl-notes" class="block text-sm font-medium text-text mb-1">Notes</label>
          <textarea id="rl-notes" name="notes" rows="2" class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" placeholder="Optional notes about this reading session..."></textarea>
        </div>
      </div>
      <div class="flex justify-end gap-3 mt-6">
        <button type="button" hx-on:click="document.getElementById('modal-backdrop').remove()" class="px-4 py-2 rounded-lg border text-text-light hover:text-text transition-colors no-underline" style="border-color: var(--color-secondary);">Cancel</button>
        <button type="submit" class="px-4 py-2 rounded-lg font-medium text-white" style="background-color: var(--color-primary);">Log Reading</button>
      </div>
    </form>
  </div>
</div>`))
	}
}

// HTMLCreateReadingLogHandler creates a reading log entry via HTMX form POST.
func HTMLCreateReadingLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		bookIDStr := r.FormValue("book_id")
		bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
		if err != nil || bookID == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid book ID")))
			return
		}

		// Handle reader name: use manual input if "other" was selected, otherwise use select value
		readerName := strings.TrimSpace(r.FormValue("reader_name"))
		if readerName == "other" {
			readerName = strings.TrimSpace(r.FormValue("reader_name_manual"))
		}
		if readerName == "" {
			var errs validation.Errors
			errs.Required("reader_name", "")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment(errs.First())))
			return
		}

		// Parse start/end pages
		var startPage, endPage *int
		if sp := strings.TrimSpace(r.FormValue("start_page")); sp != "" {
			v, err := strconv.Atoi(sp)
			if err == nil && v > 0 {
				startPage = &v
			}
		}
		if ep := strings.TrimSpace(r.FormValue("end_page")); ep != "" {
			v, err := strconv.Atoi(ep)
			if err == nil && v > 0 {
				endPage = &v
			}
		}

		entireBook := r.FormValue("entire_book") == "true"
		if entireBook {
			// If entire book, clear page numbers
			startPage = nil
			endPage = nil
		}

		// Calculate total pages
		totalPages := 0
		if startPage != nil && endPage != nil {
			totalPages = *endPage - *startPage + 1
			if totalPages < 0 {
				totalPages = 0
			}
		}

		// Parse read_at — datetime-local inputs are always in the user's local time.
		// We parse them as local time, then convert to UTC for storage.
		readAt := strings.TrimSpace(r.FormValue("read_at"))
		if readAt != "" {
			// datetime-local format: 2006-01-02T15:04
			t, err := time.ParseInLocation("2006-01-02T15:04", readAt, time.Local)
			if err != nil {
				// Try full format: 2006-01-02 15:04:05
				t, err = time.ParseInLocation("2006-01-02 15:04:05", readAt, time.Local)
				if err != nil {
					readAt = time.Now().UTC().Format("2006-01-02 15:04:05")
				} else {
					readAt = t.UTC().Format("2006-01-02 15:04:05")
				}
			} else {
				readAt = t.UTC().Format("2006-01-02 15:04:05")
			}
		} else {
			readAt = time.Now().UTC().Format("2006-01-02 15:04:05")
		}

		notes := strings.TrimSpace(r.FormValue("notes"))
		var notesPtr *string
		if notes != "" {
			notesPtr = &notes
		}

		_, err = db.Exec(
			"INSERT INTO reading_logs (book_id, start_page, end_page, total_pages, entire_book, read_at, reader_name, notes) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			bookID, startPage, endPage, totalPages, entireBook, readAt, readerName, notesPtr,
		)
		if err != nil {
			HTMXErrorResponse(w, r, http.StatusInternalServerError, "Failed to create reading log entry")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Redirect", "/reading-log")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLDeleteReadingLogHandler deletes a reading log entry via HTMX DELETE.
func HTMLDeleteReadingLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid ID")))
			return
		}

		result, err := db.Exec("DELETE FROM reading_logs WHERE id = ?", id)
		if err != nil {
			HTMXErrorResponse(w, r, http.StatusInternalServerError, "Failed to delete reading log entry")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("Reading log entry not found")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLReadingLogEditFormHandler returns a modal HTML fragment for editing a reading log entry.
// GET /reading-logs/:id/edit-form
func HTMLReadingLogEditFormHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid ID")))
			return
		}

		// Load the reading log entry with book info
		var rl models.ReadingLog
		var entireBook int
		err = db.QueryRow(`SELECT rl.id, rl.book_id, rl.start_page, rl.end_page, rl.total_pages, rl.entire_book, rl.read_at, rl.reader_name, rl.notes, rl.created_at, b.title
			FROM reading_logs rl JOIN books b ON b.id = rl.book_id WHERE rl.id = ?`, id).Scan(
			&rl.ID, &rl.BookID, &rl.StartPage, &rl.EndPage, &rl.TotalPages, &entireBook,
			&rl.ReadAt, &rl.ReaderName, &rl.Notes, &rl.CreatedAt, &rl.BookTitle,
		)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		rl.EntireBook = entireBook != 0

		// Load family members for reader dropdown
		fmRows, err := db.Query("SELECT id, name, relation FROM family_members ORDER BY name ASC")
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer fmRows.Close()

		var fmOptions string
		for fmRows.Next() {
			var fmID int64
			var name, relation string
			if err := fmRows.Scan(&fmID, &name, &relation); err == nil {
				selected := ""
				if name == rl.ReaderName {
					selected = " selected"
				}
				fmOptions += fmt.Sprintf(`<option value="%s"%s>%s (%s)</option>`, template.HTMLEscapeString(name), selected, template.HTMLEscapeString(name), template.HTMLEscapeString(relation))
			}
		}

		// Load user timezone for the datetime-local default
		var userTZ string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "user_timezone").Scan(&userTZ); err != nil {
			userTZ = "America/New_York"
		}
		if userTZ == "" {
			userTZ = "America/New_York"
		}
		tz, tzErr := time.LoadLocation(userTZ)
		if tzErr != nil {
			tz = time.Local
		}

		// Convert stored UTC time to user's local time for the datetime-local input
		localReadAt := ""
		if rl.ReadAt != "" {
			t, err := time.Parse("2006-01-02 15:04:05", rl.ReadAt)
			if err == nil {
				localReadAt = t.In(tz).Format("2006-01-02T15:04")
			}
		}

		startPageVal := ""
		if rl.StartPage != nil {
			startPageVal = strconv.Itoa(*rl.StartPage)
		}
		endPageVal := ""
		if rl.EndPage != nil {
			endPageVal = strconv.Itoa(*rl.EndPage)
		}

		maxPage := ""
		endPagePlaceholder := ""
		if rl.TotalPages > 0 {
			endPagePlaceholder = strconv.Itoa(rl.TotalPages)
		}

		// Check if there's a page_count on the book for max validation
		var pageCount *int
		_ = db.QueryRow("SELECT page_count FROM books WHERE id = ?", rl.BookID).Scan(&pageCount)
		if pageCount != nil && *pageCount > 0 {
			maxPage = fmt.Sprintf(" max=\"%d\"", *pageCount)
			if endPagePlaceholder == "" {
				endPagePlaceholder = strconv.Itoa(*pageCount)
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// #nosec G705 -- All interpolated values are escaped via template.HTMLEscapeString()
		_, _ = w.Write([]byte(`
<div id="modal-backdrop" class="modal-backdrop" hx-on:click="if(event.target===this)document.getElementById('modal-backdrop').remove()">
  <div class="modal-content modal-md p-6" role="dialog" aria-modal="true">
    <div class="flex items-center justify-between mb-4 pb-3 border-b" style="border-color: rgba(139, 69, 19, 0.1);">
      <h2 class="text-xl font-heading font-semibold text-primary">Edit Reading Log</h2>
      <button type="button" hx-on:click="document.getElementById('modal-backdrop').remove()" class="text-text-light hover:text-text transition-colors text-2xl no-underline" aria-label="Close modal">×</button>
    </div>
    <p class="text-sm text-text-light mb-4">Book: <strong class="text-text">` + template.HTMLEscapeString(rl.BookTitle) + `</strong></p>
    <form hx-put="/reading-logs/` + fmt.Sprintf("%d", id) + `" hx-target="#modal-target" hx-swap="outerHTML">
      <div class="space-y-4">
        <div id="page-fields" class="grid grid-cols-2 gap-3"` + func() string {
			if rl.EntireBook {
				return ` style="display:none"`
			} else {
				return ""
			}
		}() + `>
          <div>
            <label for="rl-start-page" class="block text-sm font-medium text-text mb-1">Start Page</label>
            <input type="number" id="rl-start-page" name="start_page" min="1"` + maxPage + ` class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" placeholder="1" value="` + template.HTMLEscapeString(startPageVal) + `">
          </div>
          <div>
            <label for="rl-end-page" class="block text-sm font-medium text-text mb-1">End Page</label>
            <input type="number" id="rl-end-page" name="end_page" min="1"` + maxPage + ` class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" placeholder="` + template.HTMLEscapeString(endPagePlaceholder) + `" value="` + template.HTMLEscapeString(endPageVal) + `">
          </div>
        </div>
        <div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" name="entire_book" value="true" class="w-4 h-4 rounded border-secondary text-primary focus:ring-primary/20" hx-on:change="var f=document.getElementById('page-fields'); if(f)f.style.display=this.checked?'none':''"` + func() string {
			if rl.EntireBook {
				return ` checked`
			} else {
				return ""
			}
		}() + `>
            <span class="text-sm text-text">Read entire book</span>
          </label>
        </div>
        <div>
          <label for="rl-read-at" class="block text-sm font-medium text-text mb-1">Date &amp; Time</label>
          <input type="datetime-local" id="rl-read-at" name="read_at" value="` + template.HTMLEscapeString(localReadAt) + `" class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="rl-reader" class="block text-sm font-medium text-text mb-1">Reader <span class="text-error">*</span></label>
          <select id="rl-reader" name="reader_name" required class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" hx-on:change="var o=document.getElementById('rl-reader-other'); if(o)o.classList.toggle('hidden',this.value!=='other'); if(this.value==='other'&&o)o.focus()">
            <option value="">Select reader...</option>` + fmOptions + `
            <option value="other"` + func() string {
			if rl.ReaderName == "other" {
				return " selected"
			} else {
				return ""
			}
		}() + `>Other (type manually)</option>
          </select>
          <input type="text" id="rl-reader-other" name="reader_name_manual" class="w-full px-3 py-2 rounded-lg border bg-surface text-sm mt-2 hidden" style="border-color: var(--color-secondary);" placeholder="Enter reader name" value="` + func() string {
			if rl.ReaderName == "other" && rl.Notes != nil {
				return template.HTMLEscapeString(*rl.Notes)
			} else {
				return ""
			}
		}() + `">
        </div>
        <div>
          <label for="rl-notes" class="block text-sm font-medium text-text mb-1">Notes</label>
          <textarea id="rl-notes" name="notes" rows="2" class="w-full px-3 py-2 rounded-lg border bg-surface text-sm" style="border-color: var(--color-secondary);" placeholder="Optional notes about this reading session...">` + func() string {
			if rl.Notes != nil {
				return template.HTMLEscapeString(*rl.Notes)
			} else {
				return ""
			}
		}() + `</textarea>
        </div>
      </div>
      <div class="flex justify-end gap-3 mt-6">
        <button type="button" hx-on:click="document.getElementById('modal-backdrop').remove()" class="px-4 py-2 rounded-lg border text-text-light hover:text-text transition-colors no-underline" style="border-color: var(--color-secondary);">Cancel</button>
        <button type="submit" class="px-4 py-2 rounded-lg font-medium text-white" style="background-color: var(--color-primary);">Save Changes</button>
      </div>
    </form>
  </div>
</div>`))
	}
}

// HTMLUpdateReadingLogHandler updates a reading log entry via HTMX form PUT.
// PUT /reading-logs/:id
func HTMLUpdateReadingLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid ID")))
			return
		}

		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		// Handle reader name: use manual input if "other" was selected, otherwise use select value
		readerName := strings.TrimSpace(r.FormValue("reader_name"))
		if readerName == "other" {
			readerName = strings.TrimSpace(r.FormValue("reader_name_manual"))
		}
		if readerName == "" {
			var errs validation.Errors
			errs.Required("reader_name", "")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment(errs.First())))
			return
		}

		// Parse start/end pages
		var startPage, endPage *int
		if sp := strings.TrimSpace(r.FormValue("start_page")); sp != "" {
			v, err := strconv.Atoi(sp)
			if err == nil && v > 0 {
				startPage = &v
			}
		}
		if ep := strings.TrimSpace(r.FormValue("end_page")); ep != "" {
			v, err := strconv.Atoi(ep)
			if err == nil && v > 0 {
				endPage = &v
			}
		}

		entireBook := r.FormValue("entire_book") == "true"
		if entireBook {
			startPage = nil
			endPage = nil
		}

		// Parse read_at — same UTC conversion logic as create
		readAt := strings.TrimSpace(r.FormValue("read_at"))
		if readAt != "" {
			t, err := time.ParseInLocation("2006-01-02T15:04", readAt, time.Local)
			if err != nil {
				t, err = time.ParseInLocation("2006-01-02 15:04:05", readAt, time.Local)
				if err != nil {
					readAt = time.Now().UTC().Format("2006-01-02 15:04:05")
				} else {
					readAt = t.UTC().Format("2006-01-02 15:04:05")
				}
			} else {
				readAt = t.UTC().Format("2006-01-02 15:04:05")
			}
		}

		notes := strings.TrimSpace(r.FormValue("notes"))
		var notesPtr *string
		if notes != "" {
			notesPtr = &notes
		}

		// Build update query dynamically
		var setClauses []string
		var args []interface{}

		if startPage != nil || endPage != nil {
			// Need to recalculate total_pages
			var sp, ep *int
			if startPage != nil {
				sp = startPage
			} else {
				var existing int
				err := db.QueryRow("SELECT start_page FROM reading_logs WHERE id = ?", id).Scan(&existing)
				if err == nil {
					sp = &existing
				}
			}
			if endPage != nil {
				ep = endPage
			} else {
				var existing int
				err := db.QueryRow("SELECT end_page FROM reading_logs WHERE id = ?", id).Scan(&existing)
				if err == nil {
					ep = &existing
				}
			}
			if sp != nil && ep != nil {
				tot := *ep - *sp + 1
				if tot < 0 {
					tot = 0
				}
				setClauses = append(setClauses, "total_pages = ?")
				args = append(args, tot)
			}
		}

		if startPage != nil {
			setClauses = append(setClauses, "start_page = ?")
			args = append(args, *startPage)
		}
		if endPage != nil {
			setClauses = append(setClauses, "end_page = ?")
			args = append(args, *endPage)
		}
		if entireBook {
			setClauses = append(setClauses, "entire_book = ?")
			args = append(args, 1)
		}
		if readAt != "" {
			setClauses = append(setClauses, "read_at = ?")
			args = append(args, readAt)
		}
		setClauses = append(setClauses, "reader_name = ?")
		args = append(args, readerName)
		if notesPtr != nil {
			setClauses = append(setClauses, "notes = ?")
			args = append(args, *notesPtr)
		}

		if len(setClauses) == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			return
		}

		setClause := strings.Join(setClauses, ", ")
		args = append(args, id)
		// #nosec G202 -- Column names are hardcoded constants, not user input
		query := "UPDATE reading_logs SET " + setClause + " WHERE id = ?"
		result, err := db.Exec(query, args...)
		if err != nil {
			HTMXErrorResponse(w, r, http.StatusInternalServerError, "Failed to update reading log entry")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("Reading log entry not found")))
			return
		}

		// Redirect back to reading log page on success
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Redirect", "/reading-log")
		w.WriteHeader(http.StatusOK)
	}
}

// --- Page Handlers ---

// RenderReadingLogPage renders the reading log page.
func RenderReadingLogPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := pages.BuildBaseContext(r, store, sessionName, db)
		ctx := pages.PageContext{BaseContext: base}

		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Load reading logs with book titles
		query := `SELECT rl.id, rl.book_id, b.title, rl.start_page, rl.end_page, rl.total_pages, rl.entire_book, rl.read_at, rl.reader_name, rl.notes, rl.created_at FROM reading_logs rl JOIN books b ON b.id = rl.book_id ORDER BY rl.read_at DESC`
		rows, err := db.Query(query)
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		logs := make([]models.ReadingLog, 0)
		for rows.Next() {
			var rl models.ReadingLog
			var entireBook int
			err := rows.Scan(&rl.ID, &rl.BookID, &rl.BookTitle, &rl.StartPage, &rl.EndPage, &rl.TotalPages, &entireBook, &rl.ReadAt, &rl.ReaderName, &rl.Notes, &rl.CreatedAt)
			if err != nil {
				HTMXError(w, r, http.StatusInternalServerError)
				return
			}
			rl.EntireBook = entireBook != 0
			logs = append(logs, rl)
		}
		if err = rows.Err(); err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		ctx.ReadingLogs = logs

		// Load family members for the "Log Reading" modal
		fmRows, err := db.Query("SELECT id, name, relation FROM family_members ORDER BY name ASC")
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer fmRows.Close()

		ctx.FamilyMembers = make([]models.FamilyMember, 0)
		for fmRows.Next() {
			var fm models.FamilyMember
			if err := fmRows.Scan(&fm.ID, &fm.Name, &fm.Relation); err == nil {
				ctx.FamilyMembers = append(ctx.FamilyMembers, fm)
			}
		}

		// Load recent books for quick logging
		bookRows, err := db.Query("SELECT id, title FROM books ORDER BY created_at DESC LIMIT 20")
		if err != nil {
			HTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer bookRows.Close()

		type bookSelect struct {
			ID    int64
			Title string
		}
		books := make([]bookSelect, 0)
		for bookRows.Next() {
			var b bookSelect
			if err := bookRows.Scan(&b.ID, &b.Title); err == nil {
				books = append(books, b)
			}
		}
		ctx.RecentBooks = books

		pages.RenderPage(w, r, tmpl, "reading-log.html", ctx)
	}
}
