package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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
