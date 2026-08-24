package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type contextKey string

const currentUserKey contextKey = "current-user"

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	var user User
	var password, created string
	err := a.db.QueryRow(`SELECT id, email, password, name, role, created_at FROM users WHERE email = ?`, input.Email).
		Scan(&user.ID, &user.Email, &password, &user.Name, &user.Role, &created)
	if err != nil || password != input.Password {
		// A07: intentionally no throttling, lockout, or progressive delay.
		a.audit(r, "authentication_failure", "", http.StatusUnauthorized, "failed login")
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials")
		return
	}
	user.CreatedAt = parseDBTime(created)

	token := newToken()
	expires := time.Now().UTC().Add(24 * time.Hour)
	_, _ = a.db.Exec(`INSERT INTO sessions(token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		token, user.ID, expires.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))

	a.audit(r, "authentication_success", user.ID, http.StatusOK, "user logged in")
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken": token,
		"expiresAt":   expires,
	})
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			a.audit(r, "authentication_failure", "", http.StatusUnauthorized, "missing bearer token")
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}

		var user User
		var expires, created string
		err := a.db.QueryRow(`
            SELECT u.id, u.email, u.name, u.role, u.created_at, s.expires_at
            FROM sessions s
            JOIN users u ON u.id = s.user_id
            WHERE s.token = ?
        `, token).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &created, &expires)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
			return
		}

		expiry, err := time.Parse(time.RFC3339Nano, expires)
		if err != nil || time.Now().UTC().After(expiry) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "expired bearer token")
			return
		}
		user.CreatedAt = parseDBTime(created)

		ctx := context.WithValue(r.Context(), currentUserKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) User {
	user, _ := r.Context().Value(currentUserKey).(User)
	return user
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	_, _ = a.db.Exec(`DELETE FROM sessions WHERE token = ?`, bearerToken(r))
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) requireAdmin(w http.ResponseWriter, r *http.Request) (User, bool) {
	user := currentUser(r)
	if user.ID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return User{}, false
	}
	if user.Role != "admin" {
		a.audit(r, "authorization_failure", user.ID, http.StatusForbidden, "admin role required")
		writeError(w, http.StatusForbidden, "forbidden", "admin role required")
		return User{}, false
	}
	return user, true
}

var _ = sql.ErrNoRows
