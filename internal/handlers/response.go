package handlers

import (
	"encoding/json"
	"net/http"
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
