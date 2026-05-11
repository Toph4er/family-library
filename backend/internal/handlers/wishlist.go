package handlers

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ListWishlistHandler returns all wishlist items
func ListWishlistHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// CreateWishlistItemHandler adds a new wishlist item
func CreateWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// UpdateWishlistItemHandler updates a wishlist item
func UpdateWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = id
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// DeleteWishlistItemHandler removes a wishlist item
func DeleteWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = id
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}

// FulfillWishlistItemHandler marks a wishlist item as fulfilled
func FulfillWishlistItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = id
		JSONResponse(w, http.StatusNotImplemented, map[string]interface{}{"error": "not implemented"})
	}
}
