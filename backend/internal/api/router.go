// Package api provides the HTTP router and route registration.
package api

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"
	"strconv"

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
	r.Post("/settings/guest-visibility/update", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateGuestVisibilityHandler(database))).ServeHTTP(w, r)
	})

	// User management (HTMX, under /settings)
	r.Get("/settings/users/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUserFormHandler(database))).ServeHTTP(w, r)
	})
	r.Get("/settings/users/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUserFormHandler(database))).ServeHTTP(w, r)
	})
	r.Post("/settings/users", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLCreateUserHandler(database))).ServeHTTP(w, r)
	})
	r.Put("/settings/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateUserHandler(database))).ServeHTTP(w, r)
	})
	r.Delete("/settings/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLDeleteUserHandler(database))).ServeHTTP(w, r)
	})

	// Family member management (HTMX, under /settings)
	r.Get("/settings/family-members/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLFamilyMemberFormHandler(database))).ServeHTTP(w, r)
	})
	r.Get("/settings/family-members/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLFamilyMemberFormHandler(database))).ServeHTTP(w, r)
	})
	r.Post("/settings/family-members", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLCreateFamilyMemberHandler(database))).ServeHTTP(w, r)
	})
	r.Put("/settings/family-members/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateFamilyMemberHandler(database))).ServeHTTP(w, r)
	})
	r.Delete("/settings/family-members/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLDeleteFamilyMemberHandler(database))).ServeHTTP(w, r)
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
	// Standalone book form pages (admin only)
	r.Get("/books/add-book", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.RenderBookFormPage(cfg.Templates["book-form"], database, authSvc.Store(), auth.SessionID, false, 0))).ServeHTTP(w, r)
	})
	r.Get("/books/{id}/edit-book", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			handlers.RenderBookFormPage(cfg.Templates["book-form"], database, authSvc.Store(), auth.SessionID, true, id).ServeHTTP(w, r)
		})).ServeHTTP(w, r)
	})

	// Book form POST handlers (admin only)
	r.Post("/books/create", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLCreateBookHandler(database))).ServeHTTP(w, r)
	})
	r.Post("/books/{id}/update", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateBookHandler(database))).ServeHTTP(w, r)
	})
	// Wishlist (open to guests for viewing; admin-only for management)
	r.Get("/wishlist", handlers.RenderWishlistPage(cfg.Templates["wishlist"], database, authSvc.Store(), auth.SessionID))
	r.Get("/wishlist/add", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.RenderWishlistFormPage(cfg.Templates["wishlist-form"], database, authSvc.Store(), auth.SessionID, false, 0))).ServeHTTP(w, r)
	})
	r.Get("/wishlist/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			handlers.RenderWishlistFormPage(cfg.Templates["wishlist-form"], database, authSvc.Store(), auth.SessionID, true, id).ServeHTTP(w, r)
		})).ServeHTTP(w, r)
	})
	r.Post("/wishlist/create", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLCreateWishlistItemHandler(database))).ServeHTTP(w, r)
	})
	r.Post("/wishlist/{id}/update", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.HTMLUpdateWishlistItemHandler(database))).ServeHTTP(w, r)
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(handlers.RenderSettingsPage(cfg.Templates["settings"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})

	// Reading log (authenticated users)
	r.Get("/reading-log", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.RenderReadingLogPage(cfg.Templates["reading-log"], database, authSvc.Store(), auth.SessionID))).ServeHTTP(w, r)
	})
	r.Get("/reading-logs/{book_id}/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.HTMLReadingLogFormHandler(database))).ServeHTTP(w, r)
	})
	r.Post("/reading-logs", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.HTMLCreateReadingLogHandler(database))).ServeHTTP(w, r)
	})
	r.Delete("/reading-logs/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(http.HandlerFunc(handlers.HTMLDeleteReadingLogHandler(database))).ServeHTTP(w, r)
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
				r.Get("/search-ol", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAuth(http.HandlerFunc(handlers.SearchOpenLibraryHandler(database))).ServeHTTP(w, r)
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

			// Author works (Open Library series inference, auth required)
			r.Get("/authors/works", func(w http.ResponseWriter, r *http.Request) {
				authSvc.RequireAuth(http.HandlerFunc(handlers.AuthorWorksHandler(database))).ServeHTTP(w, r)
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
				r.Post("/{id}/fulfill", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(http.HandlerFunc(handlers.FulfillWishlistItemHandler(database))).ServeHTTP(w, r)
				})
			})

			// Settings (admin only)
			r.Route("/settings", func(r chi.Router) {
				r.Use(authSvc.RequireAdmin)
				r.Get("/", handlers.ListSettingsHandler(database))
				r.Put("/{key}", handlers.UpdateSettingHandler(database))
			})

			// User management (admin only, under /settings)
			r.Route("/settings/users", func(r chi.Router) {
				r.Use(authSvc.RequireAdmin)
				r.Get("/", handlers.ListUsersHandler(database))
				r.Post("/", handlers.CreateUserHandler(database))
				r.Put("/{id}", handlers.UpdateUserHandler(database))
				r.Delete("/{id}", handlers.DeleteUserHandler(database))
			})

			// Family member management (admin only, under /settings)
			r.Route("/settings/family-members", func(r chi.Router) {
				r.Use(authSvc.RequireAdmin)
				r.Get("/", handlers.ListFamilyMembersHandler(database))
				r.Post("/", handlers.CreateFamilyMemberHandler(database))
				r.Put("/{id}", handlers.UpdateFamilyMemberHandler(database))
				r.Delete("/{id}", handlers.DeleteFamilyMemberHandler(database))
			})
		})
	})

	return r
}
