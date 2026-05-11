// Package api provides the HTTP router and route registration.
package api

import (
	"database/sql"
	"embed"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/handlers"
	"git.rcsmaine.com/chris/library/backend/internal/middleware"
)

//go:embed static/*
var staticFS embed.FS

// NewRouter constructs and returns the application router with all routes
// and middleware configured.
func NewRouter(database *sql.DB, authSvc *auth.Auth) http.Handler {
	r := chi.NewRouter()

	// -- Standard chi middleware --
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	// -- Custom middleware --
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.RateLimiter)

	// -- CORS --
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "https://library.rcsmaine.com"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{corsOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// -- Health check (public) --
	healthHandler := handlers.NewHealthHandler(database)
	r.Get("/health", healthHandler.Check)

	// -- API routes --
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication (login endpoints are public)
		r.Post("/auth/login", handlers.LoginHandler(authSvc))
		r.Post("/auth/guest-login", handlers.GuestLoginHandler(authSvc))
		r.Post("/auth/logout", authSvc.RequireAuth(handlers.LogoutHandler(authSvc)))
		r.Get("/auth/me", authSvc.RequireAuth(handlers.MeHandler(authSvc)))

		// Books (all require auth)
		r.Route("/books", func(r chi.Router) {
			r.Use(authSvc.RequireAuth)
			r.Get("/", handlers.ListBooksHandler(database))
			r.Get("/search", handlers.SearchBooksHandler(database))
			r.Get("/{id}", handlers.GetBookHandler(database))
			r.Post("/", authSvc.RequireAdmin(handlers.CreateBookHandler(database)))
			r.Put("/{id}", authSvc.RequireAdmin(handlers.UpdateBookHandler(database)))
			r.Delete("/{id}", authSvc.RequireAdmin(handlers.DeleteBookHandler(database)))
			r.Post("/import-isbn", authSvc.RequireAdmin(handlers.ImportISBNHandler(database)))
		})

		// Wishlist (all require auth)
		r.Route("/wishlist", func(r chi.Router) {
			r.Use(authSvc.RequireAuth)
			r.Get("/", handlers.ListWishlistHandler(database))
			r.Post("/", authSvc.RequireAdmin(handlers.CreateWishlistItemHandler(database)))
			r.Put("/{id}", authSvc.RequireAdmin(handlers.UpdateWishlistItemHandler(database)))
			r.Delete("/{id}", authSvc.RequireAdmin(handlers.DeleteWishlistItemHandler(database)))
			r.Patch("/{id}/fulfill", authSvc.RequireAdmin(handlers.FulfillWishlistItemHandler(database)))
		})

		// Settings (admin only)
		r.Route("/settings", func(r chi.Router) {
			r.Use(authSvc.RequireAdmin)
			r.Get("/", handlers.ListSettingsHandler(database))
			r.Put("/{key}", handlers.UpdateSettingHandler(database))
		})

		// Admin (admin only)
		r.Route("/admin", func(r chi.Router) {
			r.Use(authSvc.RequireAdmin)
			r.Route("/users", func(r chi.Router) {
				r.Get("/", handlers.ListUsersHandler(database))
				r.Post("/", handlers.CreateUserHandler(database))
				r.Put("/{id}", handlers.UpdateUserHandler(database))
				r.Delete("/{id}", handlers.DeleteUserHandler(database))
			})
		})
	})

	// -- Static files and SPA fallback --
	// Serve static files from embedded filesystem
	staticSubFS, err := staticFS.ReadFile("static/.gitkeep")
	if err != nil {
		// No static files yet (dev mode) — serve from disk
		serveStaticFromDisk(r)
	} else {
		_ = staticSubFS
		serveStaticEmbedded(r)
	}

	// SPA fallback: any non-API, non-static route serves index.html
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}
		// Try to serve index.html from embedded FS or disk
		serveIndexHTML(w, r)
	})

	return r
}

func serveStaticFromDisk(r chi.Router) {
	// In dev mode, serve from ./static directory
	r.FileServer("/static", http.Dir("./static"))
}

func serveStaticEmbedded(r chi.Router) {
	// In prod mode, serve from embedded filesystem
	fs := http.FS(staticFS)
	r.FileServer("/static", http.StripPrefix("/static", http.FileServer(fs)))
}

func serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	// Try embedded first, then disk
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		// Fall back to disk
		http.ServeFile(w, r, "./static/index.html")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
