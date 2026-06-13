package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"git.rcsmaine.com/chris/library/internal/models"
)

const familyMemberColumns = `id, name, relation, created_at, updated_at`

// scanner is implemented by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanFamilyMember scans a row into a FamilyMember struct.
func scanFamilyMember(s scanner) (*models.FamilyMember, error) {
	var fm models.FamilyMember
	var createdAt, updatedAt string
	err := s.Scan(&fm.ID, &fm.Name, &fm.Relation, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	fm.CreatedAt = createdAt
	fm.UpdatedAt = updatedAt
	return &fm, nil
}

// --- HTMX HTML Handlers ---

// HTMLFamilyMemberFormHandler returns a modal HTML fragment for adding/editing a family member.
func HTMLFamilyMemberFormHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		isEdit := idStr != ""

		var member models.FamilyMember
		if isEdit {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(htmlErrorFragment("Invalid ID")))
				return
			}
			row := db.QueryRow("SELECT "+familyMemberColumns+" FROM family_members WHERE id = ?", id)
			m, err := scanFamilyMember(row)
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			member = *m
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		hxVerb := "post"
		actionURL := "/settings/family-members"
		titleLabel := "Add"
		buttonLabel := "Add"
		if isEdit {
			hxVerb = "put"
			actionURL = "/settings/family-members/" + idStr
			titleLabel = "Edit"
			buttonLabel = "Save"
		}

		relationOptions := []string{"Mom", "Dad", "Daughter", "Son", "Grandma", "Grandpa", "Sister", "Brother", "Aunt", "Uncle", "Other"}

		var relOpts string
		for _, rel := range relationOptions {
			selected := ""
			if isEdit && member.Relation == rel {
				selected = "selected"
			}
			relOpts += fmt.Sprintf(`<option value="%s" %s>%s</option>`, template.HTMLEscapeString(rel), selected, template.HTMLEscapeString(rel))
		}
		// Custom relation not in standard list
		if isEdit && member.Relation != "" {
			isStandard := false
			for _, rel := range relationOptions {
				if member.Relation == rel {
					isStandard = true
					break
				}
			}
			if !isStandard {
				relOpts += fmt.Sprintf(`<option value="%s" selected>%s</option>`, template.HTMLEscapeString(member.Relation), template.HTMLEscapeString(member.Relation))
			}
		}

		// #nosec G705 -- All interpolated values are escaped via template.HTMLEscapeString()
		_, _ = w.Write([]byte(`
<div class="modal-backdrop" hx-on::click="if(event.target===this)this.remove()">
  <div class="modal-content modal-sm p-6" role="dialog" aria-modal="true">
    <div class="flex items-center justify-between mb-4 pb-3 border-b" style="border-color: rgba(139, 69, 19, 0.1);">
      <h2 class="text-xl font-heading font-semibold text-primary">` + titleLabel + ` Family Member</h2>
      <button type="button" hx-on::click="document.getElementById('modal-backdrop')?.remove()" class="text-text-light hover:text-text transition-colors text-2xl no-underline" aria-label="Close modal">×</button>
    </div>
    <form hx-` + hxVerb + `="` + actionURL + `" hx-target="#fm-form-error" hx-swap="innerHTML">
      <div class="space-y-4">
        <div>
          <label for="fm-name" class="block text-sm font-medium text-text mb-1">Name <span class="text-error">*</span></label>
          <input type="text" id="fm-name" name="name" value="` + template.HTMLEscapeString(member.Name) + `" required class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);" placeholder="e.g., Emma">
        </div>
        <div>
          <label for="fm-relation" class="block text-sm font-medium text-text mb-1">Relation <span class="text-error">*</span></label>
          <select id="fm-relation" name="relation" required class="w-full px-3 py-2 rounded-lg border bg-surface" style="border-color: var(--color-secondary);">
            <option value="">Select...</option>` + relOpts + `
          </select>
        </div>
      </div>
      <div id="fm-form-error" class="mt-4"></div>
      <div class="flex justify-end gap-3 mt-6">
        <button type="button" hx-on::click="document.getElementById('modal-backdrop')?.remove()" class="px-4 py-2 rounded-lg border text-text-light hover:text-text transition-colors no-underline" style="border-color: var(--color-secondary);">Cancel</button>
        <button type="submit" class="px-4 py-2 rounded-lg font-medium text-white" style="background-color: var(--color-primary);">` + buttonLabel + `</button>
      </div>
    </form>
  </div>
</div>`))
	}
}

// HTMLCreateFamilyMemberHandler creates a family member via HTMX form POST.
func HTMLCreateFamilyMemberHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid request")))
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		relation := strings.TrimSpace(r.FormValue("relation"))

		if name == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Name is required")))
			return
		}
		if relation == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Relation is required")))
			return
		}

		_, err := db.Exec("INSERT INTO family_members (name, relation) VALUES (?, ?)", name, relation)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to create family member")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Redirect", "/settings")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLUpdateFamilyMemberHandler updates a family member via HTMX form PUT.
func HTMLUpdateFamilyMemberHandler(db *sql.DB) http.HandlerFunc {
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

		name := strings.TrimSpace(r.FormValue("name"))
		relation := strings.TrimSpace(r.FormValue("relation"))

		if name == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Name is required")))
			return
		}
		if relation == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Relation is required")))
			return
		}

		_, err = db.Exec("UPDATE family_members SET name = ?, relation = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", name, relation, id)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to update family member")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Redirect", "/settings")
		w.WriteHeader(http.StatusOK)
	}
}

// HTMLDeleteFamilyMemberHandler deletes a family member via HTMX DELETE.
func HTMLDeleteFamilyMemberHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(htmlErrorFragment("Invalid ID")))
			return
		}

		result, err := db.Exec("DELETE FROM family_members WHERE id = ?", id)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(htmlErrorFragment("Failed to delete family member")))
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(htmlErrorFragment("Family member not found")))
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}
}
