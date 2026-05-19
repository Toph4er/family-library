// Package api provides the HTTP router and route registration.
package api

import (
	"database/sql"
	"html/template"
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

// RouterConfig holds optional dependencies for the router.
type RouterConfig struct {
	Templates *template.Template
}

// NewRouter constructs and returns the application router with all routes
// and middleware configured.
func NewRouter(database *sql.DB, authSvc *auth.Auth, cfg *RouterConfig) http.Handler {
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
		ExposedHeaders:   []string{"Link", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// -- Health check (public) --
	healthHandler := handlers.NewHealthHandler(database)
	r.Get("/health", healthHandler.Check)

	// -- Login pages (public, template-rendered) --
	if cfg != nil && cfg.Templates != nil {
		r.Get("/login", handlers.RenderLoginPage(cfg.Templates, authSvc.Store(), auth.SessionID))
		r.Get("/guest-login", handlers.RenderGuestLoginPage(cfg.Templates, authSvc.Store(), auth.SessionID))
		r.Get("/logout", handlers.RenderLogoutSuccess(cfg.Templates, authSvc))

		// -- HTML render pages --
		r.Get("/", handlers.RenderLandingPage(cfg.Templates, database, authSvc.Store(), auth.SessionID))
		r.Get("/books", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAuth(http.HandlerFunc(handlers.RenderBooksPage(cfg.Templates, database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
		})
		r.Get("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAuth(http.HandlerFunc(handlers.RenderBookDetailPage(cfg.Templates, database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
		})
		r.Get("/wishlist", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAuth(http.HandlerFunc(handlers.RenderWishlistPage(cfg.Templates, database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
		})
		r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAdmin(http.HandlerFunc(handlers.RenderAdminPage(cfg.Templates, database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
		})
		r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAdmin(http.HandlerFunc(handlers.RenderSettingsPage(cfg.Templates, database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
		})
	} else {
		// Fallback: redirect to SPA if templates aren't loaded
		r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		})
		r.Get("/guest-login", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		})
		r.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		})
	}

	// -- API routes --
	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints (no CSRF middleware)
		r.Get("/csrf", handlers.CSRFTokenHandler(authSvc))
		r.Post("/auth/login", handlers.LoginHandler(authSvc))
		r.Post("/auth/guest-login", handlers.GuestLoginHandler(authSvc))

		// CSRF-protected routes (middleware applied before any routes on this sub-mux)
		r.Route("/", func(r chi.Router) {
			r.Use(middleware.CSRFProtection(authSvc.Store(), auth.SessionID))

			// Authentication (protected endpoints)
			r.Post("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
				authSvc.RequireAuth(http.HandlerFunc(handlers.LogoutHandler(authSvc))).ServeHTTP(w, r)
			})
			r.Get("/auth/me", func(w http.ResponseWriter, r *http.Request) {
				authSvc.RequireAuth(http.HandlerFunc(handlers.MeHandler(authSvc))).ServeHTTP(w, r)
			})

			// Books (all require auth)
			r.Route("/books", func(r chi.Router) {
				r.Use(authSvc.RequireAuth)

				// Static GET routes must be registered before parameterized ones
				r.Get("/", handlers.ListBooksHandler(database))
				r.Get("/search", handlers.SearchBooksHandler(database))
				r.Get("/tags", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.GetTagsHandler(database))).ServeHTTP(w, r)
				})
				r.Get("/lookup-isbn", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.LookupISBNHandler(database))).ServeHTTP(w, r)
				})
				r.Get("/{id}", handlers.GetBookHandler(database))

				// POST routes
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.CreateBookHandler(database))).ServeHTTP(w, r)
				})
				r.Post("/import-isbn", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.ImportISBNHandler(database))).ServeHTTP(w, r)
				})

				// Parameterized PUT/DELETE
				r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.UpdateBookHandler(database))).ServeHTTP(w, r)
				})
				r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.DeleteBookHandler(database))).ServeHTTP(w, r)
				})
			})

			// Wishlist (all require auth)
			r.Route("/wishlist", func(r chi.Router) {
				r.Use(authSvc.RequireAuth)
				r.Get("/", handlers.ListWishlistHandler(database))
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.CreateWishlistItemHandler(database))).ServeHTTP(w, r)
				})
				r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.UpdateWishlistItemHandler(database))).ServeHTTP(w, r)
				})
				r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.DeleteWishlistItemHandler(database))).ServeHTTP(w, r)
				})
				r.Patch("/{id}/fulfill", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.FulfillWishlistItemHandler(database))).ServeHTTP(w, r)
				})
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
	})

	// -- Static files and SPA fallback --
	// Serve static files from ./static directory (populated by Docker build)
	staticFS := http.FS(os.DirFS("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(staticFS)))

	// SPA fallback: any non-API, non-static route serves index.html
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "./static/index.html")
	})

	return r
}
