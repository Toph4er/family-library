package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
)

// ListUsersHandler returns all users
func ListUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// CreateUserHandler creates a new admin user
func CreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username    string `json:"username"`
			Password    string `json:"password"`
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Hash password
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		// TODO: Insert user into database
		_ = hash
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// UpdateUserHandler updates a user
func UpdateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = id
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// DeleteUserHandler deletes a user
func DeleteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = id
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}
