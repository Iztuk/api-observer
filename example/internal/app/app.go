package app

import (
	"database/sql"
	"net/http"
	"time"
)

type App struct {
	db      *sql.DB
	started int64
}

func New(db *sql.DB) *App {
	return &App{
		db:      db,
		started: time.Now().Unix(),
	}
}

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)

	mux.HandleFunc("POST /auth/login", a.login)
	mux.HandleFunc("POST /auth/logout", a.requireAuth(a.logout))
	mux.HandleFunc("GET /auth/me", a.requireAuth(a.me))

	mux.HandleFunc("GET /users", a.requireAuth(a.listUsers))
	mux.HandleFunc("POST /users", a.registerUser)
	mux.HandleFunc("GET /users/{id}", a.requireAuth(a.getUser))
	mux.HandleFunc("PATCH /users/{id}", a.requireAuth(a.updateUser))
	mux.HandleFunc("DELETE /users/{id}", a.requireAuth(a.deleteUser))
	mux.HandleFunc("PUT /users/{id}/role", a.requireAuth(a.updateUserRole))

	mux.HandleFunc("GET /projects", a.requireAuth(a.listProjects))
	mux.HandleFunc("POST /projects", a.requireAuth(a.createProject))
	mux.HandleFunc("GET /projects/{id}", a.requireAuth(a.getProject))
	mux.HandleFunc("PATCH /projects/{id}", a.requireAuth(a.updateProject))
	mux.HandleFunc("DELETE /projects/{id}", a.requireAuth(a.deleteProject))

	mux.HandleFunc("GET /projects/{projectID}/tasks", a.requireAuth(a.listTasks))
	mux.HandleFunc("POST /projects/{projectID}/tasks", a.requireAuth(a.createTask))
	mux.HandleFunc("GET /tasks/{id}", a.requireAuth(a.getTask))
	mux.HandleFunc("PATCH /tasks/{id}", a.requireAuth(a.updateTask))
	mux.HandleFunc("DELETE /tasks/{id}", a.requireAuth(a.deleteTask))

	mux.HandleFunc("GET /search/users", a.requireAuth(a.searchUsers))
	mux.HandleFunc("POST /search/projects", a.requireAuth(a.searchProjects))
	mux.HandleFunc("GET /files", a.requireAuth(a.getFile))
	mux.HandleFunc("GET /reports/{name}", a.requireAuth(a.generateReport))

	// Intentionally unauthenticated for controlled A02 testing.
	mux.HandleFunc("GET /admin/debug", a.adminDebug)
	mux.HandleFunc("GET /admin/error", a.adminError)

	mux.HandleFunc("GET /admin/stats", a.requireAuth(a.adminStats))
	mux.HandleFunc("GET /admin/config", a.requireAuth(a.adminConfig))
	mux.HandleFunc("GET /audit/events", a.requireAuth(a.listAuditEvents))

	return mux
}
