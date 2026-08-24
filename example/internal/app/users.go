package app

import (
	"database/sql"
	"net/http"
	"time"
)

func (a *App) registerUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if input.Email == "" || input.Password == "" || input.Name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "email, password, and name are required")
		return
	}

	user := User{ID: newID(), Email: input.Email, Name: input.Name, Role: "standard", CreatedAt: time.Now().UTC()}
	_, err := a.db.Exec(`INSERT INTO users(id,email,password,name,role,created_at) VALUES(?,?,?,?,?,?)`,
		user.ID, user.Email, input.Password, user.Name, user.Role, user.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (a *App) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT id,email,name,role,created_at FROM users ORDER BY created_at`)
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		var created string
		if rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &created) == nil {
			u.CreatedAt = parseDBTime(created)
			users = append(users, u)
		}
	}
	writeJSON(w, http.StatusOK, users)
}

func (a *App) getUser(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	id := r.PathValue("id")
	if actor.ID != id && actor.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "cannot access another user")
		return
	}
	u, err := a.findUser(id)
	if err == sql.ErrNoRows {
		writeError(w, 404, "not_found", "user not found")
		return
	}
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	writeJSON(w, 200, u)
}

func (a *App) updateUser(w http.ResponseWriter, r *http.Request) {
	actor := currentUser(r)
	id := r.PathValue("id")
	if actor.ID != id && actor.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "cannot modify another user")
		return
	}
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	_, err := a.db.Exec(`UPDATE users SET name = COALESCE(NULLIF(?,''),name), email = COALESCE(NULLIF(?,''),email) WHERE id = ?`, input.Name, input.Email, id)
	if err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	u, err := a.findUser(id)
	if err != nil {
		writeError(w, 404, "not_found", "user not found")
		return
	}
	writeJSON(w, 200, u)
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	res, err := a.db.Exec(`DELETE FROM users WHERE id = ?`, r.PathValue("id"))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "not_found", "user not found")
		return
	}
	a.audit(r, "resource_deleted", currentUser(r).ID, 204, "user deleted")
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) updateUserRole(w http.ResponseWriter, r *http.Request) {
	// A01: intentionally missing admin authorization check. Any authenticated
	// user can change any user's role in this synthetic service.
	var input struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	if input.Role != "standard" && input.Role != "admin" {
		writeError(w, 400, "bad_request", "invalid role")
		return
	}
	_, err := a.db.Exec(`UPDATE users SET role = ? WHERE id = ?`, input.Role, r.PathValue("id"))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	u, err := a.findUser(r.PathValue("id"))
	if err != nil {
		writeError(w, 404, "not_found", "user not found")
		return
	}
	writeJSON(w, 200, u)
}

func (a *App) findUser(id string) (User, error) {
	var u User
	var created string
	err := a.db.QueryRow(`SELECT id,email,name,role,created_at FROM users WHERE id=?`, id).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &created)
	u.CreatedAt = parseDBTime(created)
	return u, err
}
