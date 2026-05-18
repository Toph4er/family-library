package auth

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"

	"git.rcsmaine.com/chris/library/backend/internal/middleware"
)

const (
	SessionID   = "library_session"
	UserIDKey   = "user_id"
	RoleKey     = "role"
	UsernameKey = "username"
	IsGuestKey  = "is_guest"
)

// Auth handles authentication operations
//
type Auth struct {
	db    *sql.DB
	store *sessions.CookieStore
}

// New creates a new Auth instance
//
func New(db *sql.DB, sessionSecret []byte) *Auth {
	key := make([]byte, 32)
	copy(key, sessionSecret)
	store := sessions.NewCookieStore(key)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   30 * 60, // 30 minutes
		HttpOnly: true,
		Secure:   true,    // will be set to false in dev
		SameSite: http.SameSiteLaxMode,
	}
	return &Auth{db: db, store: store}
}

// Store returns the session store
//
func (a *Auth) Store() *sessions.CookieStore {
	return a.store
}

// getSession retrieves the session from the request context (set by the CSRF
// middleware) or loads it from the cookie store.  Using the context session
// avoids a second cookie read and prevents save-ordering conflicts when both
// the CSRF middleware and an auth handler touch the session.
func (a *Auth) getSession(r *http.Request) (*sessions.Session, error) {
	if s := middleware.GetSessionFromContext(r); s != nil {
		return s, nil
	}
	return a.store.Get(r, SessionID)
}

// HashPassword hashes a password using bcrypt
//
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword compares a password with a hash
//
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Login authenticates an admin user and creates a session
//
func (a *Auth) Login(w http.ResponseWriter, r *http.Request, username, password string) (*User, error) {
	var user User
	err := a.db.QueryRow(
		"SELECT id, username, password_hash, role, display_name, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if !CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Create session
	session, err := a.getSession(r)
	if err != nil {
		return nil, err
	}

	session.Values[UserIDKey] = user.ID
	session.Values[RoleKey] = user.Role
	session.Values[UsernameKey] = user.Username
	session.Values[IsGuestKey] = false
	session.Options.MaxAge = 24 * 60 * 60 // 24 hours for admin

	// Don't save here — the CSRF middleware will save the shared session
	// after the handler completes, preserving both the rotated token and
	// the auth fields.  Only save if the session came from the store
	// (no CSRF middleware in the chain, e.g. login endpoint).
	if middleware.GetSessionFromContext(r) == nil {
		if err := session.Save(r, w); err != nil {
			return nil, err
		}
	}

	return &user, nil
}

// GuestLogin authenticates a guest user with shared password
//
func (a *Auth) GuestLogin(w http.ResponseWriter, r *http.Request, password string) error {
	// Get stored guest password hash from settings
	var guestHash string
	err := a.db.QueryRow("SELECT value FROM settings WHERE key = 'guest_password_hash'").Scan(&guestHash)
	if err != nil || guestHash == "" {
		return ErrGuestNotConfigured
	}

	if !CheckPassword(password, guestHash) {
		return ErrInvalidCredentials
	}

	// Create guest session
	session, err := a.getSession(r)
	if err != nil {
		return err
	}

	session.Values[IsGuestKey] = true
	session.Values[RoleKey] = "guest"
	session.Options.MaxAge = 4 * 60 * 60 // 4 hours for guest

	// Only save if not in the CSRF middleware chain.
	if middleware.GetSessionFromContext(r) == nil {
		return session.Save(r, w)
	}
	return nil
}

// Logout destroys the session
//
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := a.store.Get(r, SessionID)
	if err != nil {
		return err
	}

	session.Options.MaxAge = -1
	return session.Save(r, w)
}

// GetUserFromSession retrieves the current user from the session
//
func (a *Auth) GetUserFromSession(r *http.Request) (*SessionUser, bool) {
	session, err := a.store.Get(r, SessionID)
	if err != nil {
		return nil, false
	}

	isGuest, ok := session.Values[IsGuestKey].(bool)
	if !ok {
		return nil, false
	}

	user := &SessionUser{IsGuest: isGuest}

	if !isGuest {
		if id, ok := session.Values[UserIDKey].(int64); ok {
			user.ID = id
		}
		if role, ok := session.Values[RoleKey].(string); ok {
			user.Role = role
		}
		if username, ok := session.Values[UsernameKey].(string); ok {
			user.Username = username
		}
	}

	return user, true
}

// SessionUser represents a user from session data
//
type SessionUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	IsGuest  bool   `json:"is_guest"`
}

// User represents a user from the database
//
type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
	DisplayName  *string `json:"display_name"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Errors
var (
	ErrInvalidCredentials = &APIError{Code: http.StatusUnauthorized, Message: "Invalid credentials"}
	ErrGuestNotConfigured = &APIError{Code: http.StatusForbidden, Message: "Guest access not configured"}
	ErrNotAuthenticated   = &APIError{Code: http.StatusUnauthorized, Message: "Authentication required"}
	ErrForbidden          = &APIError{Code: http.StatusForbidden, Message: "Admin access required"}
)

// APIError represents an API error
//
type APIError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

// SeedAdminUser creates the initial admin user if none exists
//
func (a *Auth) SeedAdminUser(username, password string) error {
	var count int
	err := a.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Admin already exists
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = a.db.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')",
		username, hash,
	)
	return err
}

// SeedGuestPassword sets the initial guest password
//
func (a *Auth) SeedGuestPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(
		"UPDATE settings SET value = ? WHERE key = 'guest_password_hash'",
		hash,
	)
	return err
}

// UpdateGuestPassword updates the guest password
//
func (a *Auth) UpdateGuestPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(
		"UPDATE settings SET value = ? WHERE key = 'guest_password_hash'",
		hash,
	)
	return err
}
