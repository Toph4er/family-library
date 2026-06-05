// Package handlers provides HTTP request handlers for the application.
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler creates a new HealthHandler with the given database connection.
func NewHealthHandler(database *sql.DB) *HealthHandler {
	return &HealthHandler{db: database}
}

// healthResponse represents the JSON body returned by the health check endpoint.
type healthResponse struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Check verifies database connectivity and returns the health status.
//
// GET /health
//
// Returns:
//   - 200 with {"status":"ok"} if the database is reachable.
//   - 503 with {"status":"error","detail":"..."} if the database is unreachable.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Verify the database is reachable with a lightweight query.
	if err := h.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status: "error",
			Detail: "database unreachable",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}
