package handlers

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// JSONResponse sends a JSON response with the given status code and data
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// JSONError sends a JSON error response
func JSONError(w http.ResponseWriter, status int, message string) {
	JSONResponse(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// PaginatedResponse sends a paginated JSON response
func PaginatedResponse(w http.ResponseWriter, items interface{}, total, page, perPage int) {
	totalPages := (total + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	// Ensure items is always a non-nil slice so JSON encodes [] not null.
	// A nil slice stored in an interface{} is not nil at the interface level,
	// so we use reflection to detect it.
	rv := reflect.ValueOf(items)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		items = []interface{}{}
	}

	JSONResponse(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"items":       items,
		"total":       total,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
	})
}
