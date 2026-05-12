package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
)

// userResponse is the user data returned in API responses (without password_hash).
type userResponse struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	Role        string  `json:"role"`
	DisplayName *string `json:"display_name"`
	CreatedAt   string  `json:"created_at"`
}

// scanUserRow scans a *sql.Row into a userResponse.
func scanUserRow(row *sql.Row) (userResponse, error) {
	u := userResponse{}
	err := row.Scan(&u.ID, &u.Username, &u.Role, &u.DisplayName, &u.CreatedAt)
	return u, err
}

// scanUserRows scans a *sql.Rows (single row) into a userResponse.
func scanUserRows(rows *sql.Rows) (userResponse, error) {
	u := userResponse{}
	err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.DisplayName, &u.CreatedAt)
	return u, err
}

// isUniqueConstraintError checks if an error is a SQLite UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

// ListUsersHandler returns all users
func ListUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, username, role, display_name, created_at FROM users ORDER BY id")
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to query users")
			return
		}
		defer rows.Close()

		users := make([]userResponse, 0)
		for rows.Next() {
			u, err := scanUserRows(rows)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "Failed to scan user")
				return
			}
			users = append(users, u)
		}
		if err = rows.Err(); err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to query users")
			return
		}

		JSONResponse(w, http.StatusOK, users)
	}
}

// CreateUserHandler creates a new admin user
func CreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username    string  `json:"username"`
			Password    string  `json:"password"`
			DisplayName *string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Username == "" {
			JSONError(w, http.StatusBadRequest, "Username is required")
			return
		}

		// Hash password
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		result, err := db.Exec(
			"INSERT INTO users (username, password_hash, role, display_name) VALUES (?, ?, 'admin', ?)",
			req.Username, hash, req.DisplayName,
		)
		if err != nil {
			if isUniqueConstraintError(err) {
				JSONError(w, http.StatusConflict, "Username already exists")
				return
			}
			JSONError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to get user ID")
			return
		}

		// Fetch and return the created user
		user, err := scanUserRow(db.QueryRow(
			"SELECT id, username, role, display_name, created_at FROM users WHERE id = ?",
			id,
		))
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to retrieve created user")
			return
		}

		JSONResponse(w, http.StatusCreated, user)
	}
}

// UpdateUserHandler updates a user
func UpdateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var req struct {
			Username    *string `json:"username"`
			Password    *string `json:"password"`
			DisplayName *string `json:"display_name"`
			Role        *string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Build dynamic UPDATE from non-nil fields
		setClauses := []string{}
		args := []interface{}{}

		if req.Username != nil {
			setClauses = append(setClauses, "username = ?")
			args = append(args, *req.Username)
		}
		if req.Password != nil {
			hash, err := auth.HashPassword(*req.Password)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "Failed to hash password")
				return
			}
			setClauses = append(setClauses, "password_hash = ?")
			args = append(args, hash)
		}
		if req.DisplayName != nil {
			setClauses = append(setClauses, "display_name = ?")
			args = append(args, *req.DisplayName)
		}
		if req.Role != nil {
			setClauses = append(setClauses, "role = ?")
			args = append(args, *req.Role)
		}

		if len(setClauses) == 0 {
			JSONError(w, http.StatusBadRequest, "No fields to update")
			return
		}

		setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, id)

		query := "UPDATE users SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
		result, err := db.Exec(query, args...)
		if err != nil {
			if isUniqueConstraintError(err) {
				JSONError(w, http.StatusConflict, "Username already exists")
				return
			}
			JSONError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to check update result")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "User not found")
			return
		}

		// Fetch and return the updated user
		user, err := scanUserRow(db.QueryRow(
			"SELECT id, username, role, display_name, created_at FROM users WHERE id = ?",
			id,
		))
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to retrieve updated user")
			return
		}

		JSONResponse(w, http.StatusOK, user)
	}
}

// DeleteUserHandler deletes a user
func DeleteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to delete user")
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			JSONError(w, http.StatusInternalServerError, "Failed to check delete result")
			return
		}
		if rowsAffected == 0 {
			JSONError(w, http.StatusNotFound, "User not found")
			return
		}

		JSONResponse(w, http.StatusOK, map[string]interface{}{"success": true})
	}
}
