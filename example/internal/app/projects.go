package app

import (
	"database/sql"
	"net/http"
	"time"
)

func (a *App) listProjects(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	rows, err := a.db.Query(`SELECT id,owner_id,name,description,created_at FROM projects WHERE owner_id=? ORDER BY created_at`, user.ID)
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	defer rows.Close()
	projects := []Project{}
	for rows.Next() {
		var p Project
		var created string
		if rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &created) == nil {
			p.CreatedAt = parseDBTime(created)
			projects = append(projects, p)
		}
	}
	writeJSON(w, 200, projects)
}

func (a *App) createProject(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	if input.Name == "" {
		writeError(w, 400, "bad_request", "name is required")
		return
	}
	user := currentUser(r)
	p := Project{ID: newID(), OwnerID: user.ID, Name: input.Name, Description: input.Description, CreatedAt: time.Now().UTC()}
	_, err := a.db.Exec(`INSERT INTO projects(id,owner_id,name,description,created_at) VALUES(?,?,?,?,?)`, p.ID, p.OwnerID, p.Name, p.Description, p.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	writeJSON(w, 201, p)
}

func (a *App) getProject(w http.ResponseWriter, r *http.Request) {
	// A01: intentionally no owner/admin check. Any authenticated user that knows
	// a project ID can retrieve it.
	p, err := a.findProject(r.PathValue("id"))
	if err == sql.ErrNoRows {
		writeError(w, 404, "not_found", "project not found")
		return
	}
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	writeJSON(w, 200, p)
}

func (a *App) updateProject(w http.ResponseWriter, r *http.Request) {
	// A01: intentionally no ownership check.
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	_, err := a.db.Exec(`UPDATE projects SET name=COALESCE(NULLIF(?,''),name), description=COALESCE(NULLIF(?,''),description) WHERE id=?`, input.Name, input.Description, r.PathValue("id"))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	p, err := a.findProject(r.PathValue("id"))
	if err != nil {
		writeError(w, 404, "not_found", "project not found")
		return
	}
	writeJSON(w, 200, p)
}

func (a *App) deleteProject(w http.ResponseWriter, r *http.Request) {
	// A01: intentionally no ownership check.
	res, err := a.db.Exec(`DELETE FROM projects WHERE id=?`, r.PathValue("id"))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "not_found", "project not found")
		return
	}
	a.audit(r, "resource_deleted", currentUser(r).ID, 204, "project deleted")
	w.WriteHeader(204)
}

func (a *App) findProject(id string) (Project, error) {
	var p Project
	var created string
	err := a.db.QueryRow(`SELECT id,owner_id,name,description,created_at FROM projects WHERE id=?`, id).Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &created)
	p.CreatedAt = parseDBTime(created)
	return p, err
}
