// Package api provides the HTTP router and route registration.
package api

import (
	"html/template"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"

	"github.com/Toph4er/family-library/internal/auth"
	"github.com/Toph4er/family-library/internal/handlers"
	"github.com/Toph4er/family-library/internal/middleware"
	"github.com/Toph4er/family-library/internal/repository"
)

// RouterConfig holds optional dependencies for the router.
type RouterConfig struct {
	Templates map[string]*template.Template
}

// NewRouter constructs and returns the application router with all routes
// and middleware configured.
func NewRouter(database *sqlx.DB, authSvc *auth.Auth, cfg *RouterConfig) http.Handler {
	r := chi.NewRouter()

	bookRepo := repository.NewBookRepository(database)

	// -- Standard chi middleware --
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	// -- Custom middleware --
	r.Use(middleware.GenerateCSPNonce)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RequestLogger)
	r.Use(middleware.RateLimiter)

	// -- CORS --
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "https://example.com"
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{corsOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// -- Static files --
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/web"))))

	// -- Health check (public) --
	healthHandler := handlers.NewHealthHandler(database.DB)
	r.Get("/health", healthHandler.Check)

	// -- Page routes (template-rendered) --
	r.Get("/login", handlers.RenderLoginPage(cfg.Templates["login"], database.DB, authSvc.Store(), auth.SessionID))
	r.Get("/guest-login", handlers.RenderGuestLoginPage(cfg.Templates["guest-login"], database.DB, authSvc.Store(), auth.SessionID))
	r.Get("/logout", handlers.RenderLogoutSuccess(cfg.Templates["logout"], database.DB, authSvc))
	r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(handlers.RenderDashboardPage(cfg.Templates["dashboard"], database.DB, authSvc.Store(), auth.SessionID)).ServeHTTP(w, r)
	})

	// -- HTMX UI routes (form-encoded, return HTML fragments or HX-Redirect) --
	r.Post("/auth/login", handlers.HTMLLoginHandler(authSvc))
	r.Post("/auth/guest-login", handlers.HTMLGuestLoginHandler(authSvc))
	r.Put("/settings/update/{key}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUpdateSettingHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Post("/settings/guest-visibility/update", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUpdateGuestVisibilityHandler(database.DB)).ServeHTTP(w, r)
	})

	// User management (HTMX, under /settings)
	r.Get("/settings/users/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUserFormHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Get("/settings/users/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUserFormHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Post("/settings/users", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLCreateUserHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Put("/settings/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUpdateUserHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Delete("/settings/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLDeleteUserHandler(database.DB)).ServeHTTP(w, r)
	})

	// Family member management (HTMX, under /settings)
	r.Get("/settings/family-members/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLFamilyMemberFormHandler(database)).ServeHTTP(w, r)
	})
	r.Get("/settings/family-members/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLFamilyMemberFormHandler(database)).ServeHTTP(w, r)
	})
	r.Post("/settings/family-members", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLCreateFamilyMemberHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Put("/settings/family-members/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUpdateFamilyMemberHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Delete("/settings/family-members/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLDeleteFamilyMemberHandler(database.DB)).ServeHTTP(w, r)
	})

	// / — public landing page; authenticated users are redirected to /books
	r.Get("/", handlers.RenderLandingPage(cfg.Templates["landing"], database.DB, authSvc.Store(), auth.SessionID))

	// Protected page routes use HTML-aware middleware (redirects instead of JSON)
	r.Get("/books", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(handlers.RenderBooksPage(cfg.Templates["books"], database.DB, authSvc.Store(), auth.SessionID)).ServeHTTP(w, r)
	})
	r.Get("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(handlers.RenderBookDetailPage(cfg.Templates["book-detail"], database.DB, authSvc.Store(), auth.SessionID)).ServeHTTP(w, r)
	})
	// Standalone book form pages (admin only)
	r.Get("/books/add-book", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.RenderBookFormPage(cfg.Templates["book-form"], database.DB, authSvc.Store(), auth.SessionID, false, 0)).ServeHTTP(w, r)
	})
	r.Get("/books/{id}/edit-book", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			handlers.RenderBookFormPage(cfg.Templates["book-form"], database.DB, authSvc.Store(), auth.SessionID, true, id).ServeHTTP(w, r)
		})).ServeHTTP(w, r)
	})

	// Book form POST handlers (admin only)
	r.Post("/books/create", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLCreateBookHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Post("/books/{id}/update", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUpdateBookHandler(database.DB)).ServeHTTP(w, r)
	})
	// Wishlist (open to guests for viewing; admin-only for management)
	r.Get("/wishlist", handlers.RenderWishlistPage(cfg.Templates["wishlist"], database.DB, authSvc.Store(), auth.SessionID))
	r.Get("/wishlist/add", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.RenderWishlistFormPage(cfg.Templates["wishlist-form"], database.DB, authSvc.Store(), auth.SessionID, false, 0)).ServeHTTP(w, r)
	})
	r.Get("/wishlist/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			handlers.RenderWishlistFormPage(cfg.Templates["wishlist-form"], database.DB, authSvc.Store(), auth.SessionID, true, id).ServeHTTP(w, r)
		})).ServeHTTP(w, r)
	})
	r.Post("/wishlist/create", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLCreateWishlistItemHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Post("/wishlist/{id}/update", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLUpdateWishlistItemHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.RenderSettingsPage(cfg.Templates["settings"], database.DB, authSvc.Store(), auth.SessionID)).ServeHTTP(w, r)
	})

	// Reading log (Guests can view; admin-only for write operations)
	r.Get("/reading-log", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAuthHTML(handlers.RenderReadingLogPage(cfg.Templates["reading-log"], database.DB, authSvc.Store(), auth.SessionID)).ServeHTTP(w, r)
	})
	r.Get("/reading-logs/book-selector", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLBookSelectorHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Get("/reading-logs/{book_id}/new-form", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLReadingLogFormHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Post("/reading-logs", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLCreateReadingLogHandler(database.DB)).ServeHTTP(w, r)
	})
	r.Delete("/reading-logs/{id}", func(w http.ResponseWriter, r *http.Request) {
		authSvc.RequireAdminHTML(handlers.HTMLDeleteReadingLogHandler(database.DB)).ServeHTTP(w, r)
	})

	// -- API routes --
	r.Route("/api/v1", func(r chi.Router) {
		// Theme CSS (public, returns CSS override block for instant theme switching)
		r.Get("/theme/{id}/css", handlers.ThemeCSSHandler())

		// ISBN lookup (admin, returns JSON metadata for client-side form population)
		r.Get("/books/lookup-isbn", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAdmin(handlers.LookupISBNHandler(database.DB)).ServeHTTP(w, r)
		})

		// Rate child (admin, updates child_rating for a book)
		r.Post("/books/rate-child", func(w http.ResponseWriter, r *http.Request) {
			authSvc.RequireAdmin(handlers.RateChildHandler(database.DB)).ServeHTTP(w, r)
		})

		// CSRF-protected routes (middleware applied before any routes on this sub-mux)
		r.Route("/", func(r chi.Router) {
			r.Use(middleware.CSRFProtection(authSvc.Store(), auth.SessionID))

			// Book delete (HTMX hx-delete, returns JSON with success message)
			r.Delete("/books/{id}", func(w http.ResponseWriter, r *http.Request) {
				authSvc.RequireAdmin(handlers.DeleteBookHandler(bookRepo)).ServeHTTP(w, r)
			})

			// Wishlist ISBN lookup (auth, returns JSON metadata for client-side form population)
			r.Get("/wishlist/lookup-isbn", func(w http.ResponseWriter, r *http.Request) {
				authSvc.RequireAdmin(handlers.LookupISBNHandler(database.DB)).ServeHTTP(w, r)
			})

			// Wishlist (all require auth)
			r.Route("/wishlist", func(r chi.Router) {
				r.Use(authSvc.RequireAuth)
				r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(handlers.DeleteWishlistItemHandler(database.DB)).ServeHTTP(w, r)
				})
				r.Post("/{id}/fulfill", func(w http.ResponseWriter, r *http.Request) {
					authSvc.RequireAdmin(handlers.FulfillWishlistItemHandler(database.DB)).ServeHTTP(w, r)
				})
			})
		})
	})

	return r
}
