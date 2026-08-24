package app

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

func (a *App) searchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, 400, "bad_request", "q is required")
		return
	}

	// A05: intentionally vulnerable SQL construction for this disposable test DB.
	// It is limited to a SELECT against the local users table.
	query := fmt.Sprintf("SELECT id,email,name,role,created_at FROM users WHERE name LIKE '%%%s%%'", q)
	rows, err := a.db.Query(query)
	if err != nil {
		a.audit(r, "suspicious_input", currentUser(r).ID, 400, "search query caused database error")
		writeError(w, 400, "bad_request", err.Error())
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
	writeJSON(w, 200, users)
}

func (a *App) searchProjects(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Query  string `json:"query"`
		Owner  string `json:"owner"`
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	// Body is deliberately accepted as opaque search input so API Observer can
	// apply body-field/pattern rules without executing commands.
	rows, err := a.db.Query(`SELECT id,owner_id,name,description,created_at FROM projects WHERE name LIKE ?`, `%`+input.Query+`%`)
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		var p Project
		var c string
		if rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &c) == nil {
			p.CreatedAt = parseDBTime(c)
			out = append(out, p)
		}
	}
	writeJSON(w, 200, out)
}

func (a *App) getFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, 400, "bad_request", "path is required")
		return
	}
	// No host filesystem access: return a synthetic result while preserving the
	// path in observed HTTP traffic for traversal-rule testing.
	writeJSON(w, 200, map[string]string{"path": path, "content": "synthetic file contents"})
}

func (a *App) generateReport(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	// No shell execution. The path value is echoed safely for command-pattern tests.
	writeJSON(w, 200, map[string]any{"name": name, "generatedAt": time.Now().UTC(), "content": "synthetic report"})
}

func (a *App) adminDebug(w http.ResponseWriter, r *http.Request) {
	// A02: intentionally unauthenticated information disclosure.
	w.Header().Set("X-Powered-By", "Observer-Test-Service/1.0")
	w.Header().Set("Server", "observer-test-go")
	writeJSON(w, 200, map[string]any{
		"version":         "1.0.0-test",
		"environment":     "development",
		"runtime":         runtime.Version(),
		"databasePath":    "./test.db",
		"internalService": "http://internal-api.local:9000",
		"debugEnabled":    true,
	})
}

func (a *App) adminError(w http.ResponseWriter, r *http.Request) {
	// A02: intentionally verbose synthetic error content.
	writeJSON(w, 500, map[string]any{
		"error":     "internal_error",
		"message":   "database query failed",
		"exception": "sql: simulated connection failure",
		"stack":     "handlers/admin.go:42 -> database/query.go:18",
		"query":     "SELECT * FROM users WHERE email = ?",
		"debug":     map[string]any{"environment": "development", "component": "user-store"},
	})
}

func (a *App) adminStats(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	var users, projects, tasks int
	a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	a.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&projects)
	a.db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&tasks)
	writeJSON(w, 200, map[string]int{"users": users, "projects": projects, "tasks": tasks})
}
func (a *App) adminConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, 200, map[string]any{"environment": "development", "databaseHost": "local", "databaseName": "test.db", "debugEnabled": true, "logLevel": "debug"})
}

var _ = strings.Contains
