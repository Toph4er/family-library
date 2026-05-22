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
