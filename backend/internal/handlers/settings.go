package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ListSettingsHandler returns all settings
func ListSettingsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Query settings table, exclude sensitive keys for non-admin
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// UpdateSettingHandler updates a single setting
func UpdateSettingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")
		_ = key

		var req struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// TODO: Update setting in database
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}
