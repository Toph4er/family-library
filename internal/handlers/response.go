package handlers

import (
	"encoding/json"
	"html/template"
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

// HTMXError sends a 4xx/5xx response for HTMX requests.
// HTMX only cares about the status code — the body is ignored
// unless hx-push-url or swap options are configured to render it.
func HTMXError(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

// HTMXErrorResponse sends a status code with an error message body.
// Used when the client may display the message (e.g., via hx-on::after-request).
func HTMXErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(htmlErrorFragment(message)))
}

// htmlErrorFragment returns a styled HTML error div for HTMX swapping.
func htmlErrorFragment(message string) string {
	return "<div class=\"p-3 rounded-lg bg-error/10 border border-error/20 text-error text-sm\" role=\"alert\">" + template.HTMLEscapeString(message) + "</div>"
}
