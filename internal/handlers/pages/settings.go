package pages

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/Toph4er/family-library/internal/models"
	"github.com/Toph4er/family-library/internal/theme"
)

// RenderSettingsPage renders the settings page (admin required).
func RenderSettingsPage(tmpl *template.Template, db *sql.DB, store *sessions.CookieStore, sessionName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := buildBaseContext(r, store, sessionName, db)
		ctx := pageContext{BaseContext: base}

		// Defense-in-depth: reject unauthenticated or non-admin requests before querying the DB.
		if !ctx.IsAuthenticated {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if !ctx.IsAdmin {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// Load settings
		rows, err := db.Query("SELECT key, value FROM settings ORDER BY key ASC")
		if err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		settings := make(map[string]string)
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				renderHTMXError(w, r, http.StatusInternalServerError)
				return
			}
			if _, sensitive := SensitiveKeys[key]; sensitive {
				continue
			}
			settings[key] = value
		}
		if err = rows.Err(); err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		ctx.Settings = settings

		// Load users
		userRows, err := db.Query("SELECT id, username, role, display_name, created_at FROM users ORDER BY id")
		if err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer userRows.Close()

		users := make([]map[string]interface{}, 0)
		for userRows.Next() {
			var id int64
			var username, role, createdAt string
			var displayName *string
			if err := userRows.Scan(&id, &username, &role, &displayName, &createdAt); err != nil {
				renderHTMXError(w, r, http.StatusInternalServerError)
				return
			}
			users = append(users, map[string]interface{}{
				"id":           id,
				"username":     username,
				"role":         role,
				"display_name": derefString(displayName),
				"created_at":   createdAt,
			})
		}
		if err = userRows.Err(); err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		ctx.Users = users

		// Load family members
		fmRows, err := db.Query("SELECT id, name, relation, created_at, updated_at FROM family_members ORDER BY name ASC")
		if err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		defer fmRows.Close()

		familyMembers := make([]models.FamilyMember, 0)
		for fmRows.Next() {
			var fm models.FamilyMember
			var createdAt, updatedAt string
			if err := fmRows.Scan(&fm.ID, &fm.Name, &fm.Relation, &createdAt, &updatedAt); err != nil {
				renderHTMXError(w, r, http.StatusInternalServerError)
				return
			}
			fm.CreatedAt = createdAt
			fm.UpdatedAt = updatedAt
			familyMembers = append(familyMembers, fm)
		}
		if err = fmRows.Err(); err != nil {
			renderHTMXError(w, r, http.StatusInternalServerError)
			return
		}
		ctx.FamilyMembers = familyMembers

		// Load default guest visibility
		var defaultVisibility string
		if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", "default_guest_visibility").Scan(&defaultVisibility); err == nil {
			ctx.DefaultGuestVisibility = make(map[string]bool)
			_ = json.Unmarshal([]byte(defaultVisibility), &ctx.DefaultGuestVisibility)
		}

		// Load available themes
		ctx.AvailableThemes = theme.AvailableThemes()

		// Build theme colors map for server-side JS rendering
		ctx.ThemeColorsJSON = buildThemeColorsJSON(theme.AvailableThemes())

		renderPage(w, r, tmpl, "settings.html", ctx)
	}
}
