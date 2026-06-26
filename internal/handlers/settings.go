package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/theme"
)

// HTMLUpdateSettingHandler updates a single setting by key via HTMX (form-encoded, returns HTML).
// On success: returns an HTML fragment with a "✓ Saved" toast trigger.
// On failure: returns an HTML error fragment.
//
// PUT /settings/update/:key
func HTMLUpdateSettingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")

		// Block updates to sensitive keys
		if _, sensitive := pages.SensitiveKeys[key]; sensitive {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(htmlErrorFragment("Cannot update this setting via this endpoint")))
			return
		}

		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		val := r.FormValue("value")

		result, err := db.Exec(
			"UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?",
			val, key,
		)
		if err != nil {
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to update setting")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to verify update")
			return
		}
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("Setting not found")))
			return
		}

		// Success — trigger a toast via HTMX header
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", `{"settingsUpdated": true}`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<span class=\"text-success text-sm ml-2\">✓ Saved</span>"))
	}
}

// ThemeCSSHandler returns the CSS override block for a given theme ID.
// The CSSOverrideBlock returns template.HTML with <style> tags.
// Strip them so the API returns raw CSS (Content-Type: text/css).
// GET /api/v1/theme/:id/css
func ThemeCSSHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		t := theme.GetThemeByID(id)
		css := string(t.CSSOverrideBlock())
		// Trim whitespace, then remove <style> and </style> tags
		css = strings.TrimSpace(css)
		css = strings.TrimPrefix(css, "<style>")
		css = strings.TrimSuffix(css, "</style>")
		css = strings.TrimSpace(css)
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		// #nosec G705 -- css is sanitized: <style> tags stripped,
		// CSSOverrideBlock() returns static theme CSS, not user input.
		fmt.Fprint(w, css)
	}
}

// HTMLUpdateGuestVisibilityHandler toggles a single field in the default_guest_visibility
// setting.  Expects a JSON body {"field": "isbn", "visible": true}.
//
// POST /settings/guest-visibility/update
func HTMLUpdateGuestVisibilityHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Field   string `json:"field"`
			Visible bool   `json:"visible"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		// Read current visibility blob
		var blob string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "default_guest_visibility").Scan(&blob); err != nil {
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to read settings")
			return
		}

		var visibility map[string]bool
		if err := json.Unmarshal([]byte(blob), &visibility); err != nil {
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to parse settings")
			return
		}

		visibility[req.Field] = req.Visible

		newBlob, err := json.Marshal(visibility)
		if err != nil {
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to encode settings")
			return
		}

		_, err = db.Exec(
			"UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?",
			string(newBlob), "default_guest_visibility",
		)
		if err != nil {
			HTMXErrorResponse(w, http.StatusInternalServerError, "Failed to update settings")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", `{"visibilityUpdated": true}`)
		w.WriteHeader(http.StatusOK)
	}
}
