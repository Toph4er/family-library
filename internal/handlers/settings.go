package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/internal/theme"
)

// sensitiveKeys lists setting keys that must never be exposed in list responses.
var sensitiveKeys = map[string]struct{}{
	"guest_password_hash": {},
}

// ListSettingsHandler returns all settings as a key-value map.
// Sensitive keys (guest_password_hash) are excluded from the response.
//
// GET /api/v1/settings/
func ListSettingsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT key, value FROM settings ORDER BY key ASC")
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to query settings")
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				JSONError(w, http.StatusInternalServerError, "Failed to read setting")
				return
			}
			if _, sensitive := sensitiveKeys[key]; sensitive {
				continue
			}
			settings[key] = value
		}
		if err := rows.Err(); err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to read settings")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data":    settings,
		})
	}
}

// UpdateSettingHandler updates a single setting by key.
// Returns 403 for sensitive keys (guest_password_hash), 404 if the key doesn't exist.
//
// PUT /api/v1/settings/:key
// Body: {"value": "new_value"}
func UpdateSettingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")

		// Block updates to sensitive keys
		if _, sensitive := sensitiveKeys[key]; sensitive {
			JSONError(w, http.StatusForbidden, "Cannot update this setting via this endpoint")
			return
		}

		var req struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		result, err := db.Exec(
			"UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?",
			req.Value, key,
		)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to update setting")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to verify update")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "Setting not found")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"data": map[string]string{
				"key":   key,
				"value": req.Value,
			},
		})
	}
}

// HTMLUpdateSettingHandler updates a single setting by key via HTMX (form-encoded, returns HTML).
// On success: returns an HTML fragment with a "✓ Saved" toast trigger.
// On failure: returns an HTML error fragment.
//
// PUT /settings/update/:key
func HTMLUpdateSettingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")

		// Block updates to sensitive keys
		if _, sensitive := sensitiveKeys[key]; sensitive {
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
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to update setting")))
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to verify update")))
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
		// Remove <style> and </style> tags
		css = strings.TrimPrefix(css, "<style>")
		css = strings.TrimSuffix(css, "</style>")
		css = strings.TrimSpace(css)
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
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
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to read settings")))
			return
		}

		var visibility map[string]bool
		if err := json.Unmarshal([]byte(blob), &visibility); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to parse settings")))
			return
		}

		visibility[req.Field] = req.Visible

		newBlob, err := json.Marshal(visibility)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to encode settings")))
			return
		}

		_, err = db.Exec(
			"UPDATE settings SET value = ?, updated_at = CURRENT_TIMESTAMP WHERE key = ?",
			string(newBlob), "default_guest_visibility",
		)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to update settings")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", `{"visibilityUpdated": true}`)
		w.WriteHeader(http.StatusOK)
	}
}
