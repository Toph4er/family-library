// Package api provides the HTTP router and route registration.
package api

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"git.rcsmaine.com/chris/library/backend/internal/auth"
	"git.rcsmaine.com/chris/library/backend/internal/handlers"
	"git.rcsmaine.com/chris/library/backend/internal/middleware"
)

// RouterConfig holds optional dependencies for the router.
type RouterConfig struct {
	Templates map[string]*template.Template
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

	// -- Page routes (template-rendered) --
	r.Get("/login", handlers.RenderLoginPage(cfg.Templates["login"], authSvc.Store(), auth.SessionID))
	r.Get("/guest-login", handlers.RenderGuestLoginPage(cfg.Templates["guest-login"], authSvc.Store(), auth.SessionID))
	r.Get("/logout", handlers.RenderLogoutSuccess(cfg.Templates["logout"], authSvc))

	// -- HTMX UI routes (form-encoded, return HTML fragments or HX-Redirect) --
	r.Post("/auth/login", handlers.HTMLLoginHandler(authSvc))
	r.Post("/auth/guest-login", handlers.HTMLGuestLoginHandler(authSvc))
	r.Put("/settings/update/{key}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateSettingHandler(database))).ServeHTTP(w, r)
	})

	// Admin user management (HTMX)
	r.Get("/admin/users/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUserFormHandler(database))).ServeHTTP(w, r)
	})
	r.Get("/admin/users/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUserFormHandler(database))).ServeHTTP(w, r)
	})
	r.Post("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLCreateUserHandler(database))).ServeHTTP(w, r)
	})
	r.Put("/admin/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateUserHandler(database))).ServeHTTP(w, r)
	})
	r.Delete("/admin/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLDeleteUserHandler(database))).ServeHTTP(w, r)
	})

	// / — public landing page; authenticated users are redirected to /books
	r.Get("/", handlers.RenderLandingPage(cfg.Templates["landing"], database, authSvc.Store(), auth.SessionID))

	// Protected page routes use HTML-aware middleware (redirects instead of JSON)
	r.Get("/books", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.RenderBooksPage(cfg.Templates["books"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})
	r.Get("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.RenderBookDetailPage(cfg.Templates["book-detail"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})
	r.Get("/books/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLBookFormHandler(cfg.Templates["book-form"], database))).ServeHTTP(w, r)
	})
	r.Get("/books/{id}/edit-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLBookFormHandler(cfg.Templates["book-form"], database))).ServeHTTP(w, r)
	})
	r.Get("/wishlist", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.RenderWishlistPage(cfg.Templates["wishlist"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.RenderAdminPage(cfg.Templates["admin"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.RenderSettingsPage(cfg.Templates["settings"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})

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

	return r
}
