package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/internal/auth"
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

// isUniqueConstraintError checks if an error is a SQLite UNIQUE constraint violation.
func isUniqueConstraintError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

// HTMLUserFormHandler returns a modal HTML fragment for creating or editing a user.
// For new users: GET /settings/users/new-form
// For editing: GET /settings/users/{id}/edit
func HTMLUserFormHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		isEdit := idStr != ""

		var user userResponse
		if isEdit {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(htmlErrorFragment("Invalid user ID")))
				return
			}
			row := db.QueryRow(
				"SELECT id, username, role, display_name, created_at FROM users WHERE id = ?",
				id,
			)
			user, err = scanUserRow(row)
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(htmlErrorFragment("User not found")))
				return
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`
<div class="modal-backdrop" hx-on::click="if(event.target===this)this.remove()">
  <div class="modal-content modal-sm p-6" role="dialog" aria-modal="true">
    <div class="flex items-center justify-between mb-4 pb-3 border-b" style="border-color: rgba(139, 69, 19, 0.1);">
      <h2 class="text-xl font-heading font-semibold text-primary">` + (func() string {
			if isEdit {
				return "Edit User"
			}
			return "Add User"
		})() + `</h2>
      <button type="button" hx-on::click="document.getElementById('modal-backdrop')?.remove()" class="text-text-light hover:text-text transition-colors text-2xl no-underline" aria-label="Close modal">×</button>
    </div>
    <form hx-` + (func() string {
			if isEdit {
				return `put="/settings/users/` + idStr + `"`
			}
			return `post="/settings/users"`
		})() + ` hx-target="#user-form-error" hx-swap="innerHTML">
      <div class="space-y-4">
        <div>
          <label for="user-username" class="block text-sm font-medium text-text mb-1">Username <span class="text-error">*</span></label>
          <input type="text" id="user-username" name="username" value="` + template.HTMLEscapeString(user.Username) + `" required class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);" ` + (func() string {
			if isEdit {
				return `readonly`
			}
			return ``
		})() + `>
        </div>
        <div>
          <label for="user-password" class="block text-sm font-medium text-text mb-1">Password ` + (func() string {
			if isEdit {
				return `(leave blank to keep current)`
			}
			return `<span class="text-error">*</span>`
		})() + `</label>
          <input type="password" id="user-password" name="password" class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);" ` + (func() string {
			if !isEdit {
				return `required`
			}
			return ``
		})() + `>
        </div>
        <div>
          <label for="user-display-name" class="block text-sm font-medium text-text mb-1">Display Name</label>
          <input type="text" id="user-display-name" name="display_name" value="` + template.HTMLEscapeString(func() string {
			if user.DisplayName != nil {
				return *user.DisplayName
			}
			return ""
		}()) + `" class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
        </div>
        <div>
          <label for="user-role" class="block text-sm font-medium text-text mb-1">Role</label>
          <input type="hidden" id="user-role" name="role" value="admin">
          <span class="w-full px-3 py-2.5 rounded-lg border bg-surface block text-sm" style="border-color: var(--color-secondary);">Admin</span>
        </div>
      </div>
      <div id="user-form-error" class="mt-4"></div>
      <div class="flex justify-end gap-3 mt-6">
        <button type="button" hx-on::click="document.getElementById('modal-backdrop')?.remove()" class="px-4 py-2 rounded-lg border text-text-light hover:text-text transition-colors no-underline" style="border-color: var(--color-secondary);">Cancel</button>
        <button type="submit" class="px-4 py-2 rounded-lg font-medium text-white" style="background-color: var(--color-primary);">` + (func() string {
			if isEdit {
				return "Save Changes"
			}
			return "Add User"
		})() + `</button>
      </div>
    </form>
  </div>
</div>`))
	}
}

// HTMLCreateUserHandler creates a new admin user via HTMX (form-encoded).
// On success: returns a success toast trigger.
// On failure: returns an HTML error fragment.
func HTMLCreateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")
		displayName := r.FormValue("display_name")

		if username == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Username is required")))
			return
		}
		if password == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Password is required")))
			return
		}

		hash, err := auth.HashPassword(password)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to hash password")))
			return
		}

		var displayNamePtr *string
		if displayName != "" {
			displayNamePtr = &displayName
		}

		_, err = db.Exec(
			"INSERT INTO users (username, password_hash, role, display_name) VALUES (?, ?, 'admin', ?)",
			username, hash, displayNamePtr,
		)
		if err != nil {
			if isUniqueConstraintError(err) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(htmlErrorFragment("Username already exists")))
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to create user")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", `{"userCreated": true}`)
		w.Header().Set("HX-Redirect", "/settings")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLUpdateUserHandler updates a user via HTMX (form-encoded).
// On success: returns a success toast trigger.
// On failure: returns an HTML error fragment.
func HTMLUpdateUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid user ID")))
			return
		}

		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		displayName := r.FormValue("display_name")
		role := r.FormValue("role")

		var fields []updateField
		if displayName != "" {
			fields = append(fields, updateField{column: "display_name", value: displayName})
		}
		if role != "" {
			fields = append(fields, updateField{column: "role", value: role})
		}

		if len(fields) == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("No fields to update")))
			return
		}

		fields = append(fields, updateField{column: "updated_at", value: "CURRENT_TIMESTAMP"})

		setClause, args := buildUpdateClauses(fields)
		args = append(args, id)

		// #nosec G202 -- SET clauses are hardcoded column names, not from user input
		query := "UPDATE users SET " + setClause + " WHERE id = ?"
		result, err := db.Exec(query, args...)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to update user")))
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to check update result")))
			return
		}
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("User not found")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", `{"userUpdated": true}`)
		w.Header().Set("HX-Redirect", "/settings")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLDeleteUserHandler deletes a user via HTMX.
// On success: returns empty string (HTMX removes the row via outerHTML swap).
// On failure: returns an HTML error fragment.
func HTMLDeleteUserHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid user ID")))
			return
		}

		result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to delete user")))
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to check delete result")))
			return
		}
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("User not found")))
			return
		}

		// Return empty — HTMX's outerHTML swap on closest tr will remove the row
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}
}
