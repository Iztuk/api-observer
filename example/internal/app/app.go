package app

import (
	"database/sql"
	"net/http"
	"time"
)

type App struct {
	Mux     *http.ServeMux
	db      *sql.DB
	started int64
}

func New(mux *http.ServeMux, db *sql.DB) *App {
	return &App{
		Mux:     mux,
		db:      db,
		started: time.Now().Unix(),
	}
}

func (a *App) Handler() {
	a.Mux.HandleFunc("GET /health", a.health)

	a.Mux.HandleFunc("POST /auth/login", a.login)
	a.Mux.HandleFunc("POST /auth/logout", a.requireAuth(a.logout))
	a.Mux.HandleFunc("GET /auth/me", a.requireAuth(a.me))

	a.Mux.HandleFunc("GET /users", a.requireAuth(a.listUsers))
	a.Mux.HandleFunc("POST /users", a.registerUser)
	a.Mux.HandleFunc("GET /users/{id}", a.requireAuth(a.getUser))
	a.Mux.HandleFunc("PATCH /users/{id}", a.requireAuth(a.updateUser))
	a.Mux.HandleFunc("DELETE /users/{id}", a.requireAuth(a.deleteUser))
	a.Mux.HandleFunc("PUT /users/{id}/role", a.requireAuth(a.updateUserRole))

	a.Mux.HandleFunc("GET /projects", a.requireAuth(a.listProjects))
	a.Mux.HandleFunc("POST /projects", a.requireAuth(a.createProject))
	a.Mux.HandleFunc("GET /projects/{id}", a.requireAuth(a.getProject))
	a.Mux.HandleFunc("PATCH /projects/{id}", a.requireAuth(a.updateProject))
	a.Mux.HandleFunc("DELETE /projects/{id}", a.requireAuth(a.deleteProject))

	a.Mux.HandleFunc("GET /projects/{projectID}/tasks", a.requireAuth(a.listTasks))
	a.Mux.HandleFunc("POST /projects/{projectID}/tasks", a.requireAuth(a.createTask))
	a.Mux.HandleFunc("GET /tasks/{id}", a.requireAuth(a.getTask))
	a.Mux.HandleFunc("PATCH /tasks/{id}", a.requireAuth(a.updateTask))
	a.Mux.HandleFunc("DELETE /tasks/{id}", a.requireAuth(a.deleteTask))

	a.Mux.HandleFunc("GET /search/users", a.requireAuth(a.searchUsers))
	a.Mux.HandleFunc("POST /search/projects", a.requireAuth(a.searchProjects))
	a.Mux.HandleFunc("GET /files", a.requireAuth(a.getFile))
	a.Mux.HandleFunc("GET /reports/{name}", a.requireAuth(a.generateReport))

	// Intentionally unauthenticated for controlled A02 testing.
	a.Mux.HandleFunc("GET /admin/debug", a.adminDebug)
	a.Mux.HandleFunc("GET /admin/error", a.adminError)

	a.Mux.HandleFunc("GET /admin/stats", a.requireAuth(a.adminStats))
	a.Mux.HandleFunc("GET /admin/config", a.requireAuth(a.adminConfig))
	a.Mux.HandleFunc("GET /audit/events", a.requireAuth(a.listAuditEvents))
}
