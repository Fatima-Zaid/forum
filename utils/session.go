package utils

import (
	"database/sql"
	"net/http"
	"time"

	"forum/database"
	"forum/models"

	"github.com/google/uuid"
)

const (
	SessionCookieName = "session_token"
	SessionDuration    = 24 * time.Hour
)

// CreateSession generates a new UUID session for the given user, removing
// any existing sessions for that user first so each user only ever has one
// active login session at a time, then persists it to the sessions table.
func CreateSession(userID int) (*models.Session, error) {
	if _, err := database.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return nil, err
	}

	sessionID := uuid.NewString()
	expiresAt := time.Now().Add(SessionDuration)

	_, err := database.DB.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		sessionID, userID, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &models.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

// SetSessionCookie writes the session cookie to the response, matching the
// session's expiration so the browser drops it automatically once expired.
// Secure is set based on the request: true when served over HTTPS (or behind
// a TLS-terminating proxy that sets X-Forwarded-Proto), false for plain HTTP
// so local/Docker testing over http:// still works.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, session *models.Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// isHTTPS reports whether the incoming request arrived over HTTPS, either
// directly or via a reverse proxy that sets X-Forwarded-Proto.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

// ClearSessionCookie removes the session cookie from the browser.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// GetSessionFromRequest reads the session cookie (if any) and looks up the
// matching row in the sessions table. Returns (nil, nil) if there's no
// cookie, no matching row, or the session has expired — none of these are
// treated as errors, just "not logged in". A non-nil error means something
// actually went wrong (e.g. a DB failure).
func GetSessionFromRequest(r *http.Request) (*models.Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, nil // no cookie present -> not logged in
	}

	var s models.Session
	row := database.DB.QueryRow(
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`,
		cookie.Value,
	)
	if err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // stale/unknown cookie
		}
		return nil, err
	}

	if s.IsExpired() {
		// clean up the expired row so it doesn't linger in the table
		database.DB.Exec(`DELETE FROM sessions WHERE id = ?`, s.ID)
		return nil, nil
	}

	return &s, nil
}

// GetUserFromSession resolves the logged-in user (if any) for the current
// request. Returns (nil, nil) when there's no valid session.
func GetUserFromSession(r *http.Request) (*models.User, error) {
	session, err := GetSessionFromRequest(r)
	if err != nil || session == nil {
		return nil, err
	}

	var u models.User
	row := database.DB.QueryRow(
		`SELECT id, username, email, password_hash, created_at FROM users WHERE id = ?`,
		session.UserID,
	)
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

// DestroySession deletes the current session row (if any) and clears the cookie.
func DestroySession(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		database.DB.Exec(`DELETE FROM sessions WHERE id = ?`, cookie.Value)
	}
	ClearSessionCookie(w)
}