package app

import (
	"database/sql"
	"net/http"
	"time"
)

func (a *App) listTasks(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	rows, err := a.db.Query(`SELECT id,project_id,title,description,status,created_at FROM tasks WHERE project_id=? ORDER BY created_at`, projectID)
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	defer rows.Close()
	tasks := []Task{}
	for rows.Next() {
		var t Task
		var c string
		if rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &c) == nil {
			t.CreatedAt = parseDBTime(c)
			tasks = append(tasks, t)
		}
	}
	writeJSON(w, 200, tasks)
}

func (a *App) createTask(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	if input.Title == "" {
		writeError(w, 400, "bad_request", "title is required")
		return
	}
	if input.Status == "" {
		input.Status = "open"
	}
	t := Task{ID: newID(), ProjectID: r.PathValue("projectID"), Title: input.Title, Description: input.Description, Status: input.Status, CreatedAt: time.Now().UTC()}
	_, err := a.db.Exec(`INSERT INTO tasks(id,project_id,title,description,status,created_at) VALUES(?,?,?,?,?,?)`, t.ID, t.ProjectID, t.Title, t.Description, t.Status, t.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	writeJSON(w, 201, t)
}

func (a *App) getTask(w http.ResponseWriter, r *http.Request) {
	t, err := a.findTask(r.PathValue("id"))
	if err == sql.ErrNoRows {
		writeError(w, 404, "not_found", "task not found")
		return
	}
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	writeJSON(w, 200, t)
}
func (a *App) updateTask(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	_, err := a.db.Exec(`UPDATE tasks SET title=COALESCE(NULLIF(?,''),title),description=COALESCE(NULLIF(?,''),description),status=COALESCE(NULLIF(?,''),status) WHERE id=?`, input.Title, input.Description, input.Status, r.PathValue("id"))
	if err != nil {
		writeError(w, 400, "bad_request", err.Error())
		return
	}
	t, err := a.findTask(r.PathValue("id"))
	if err != nil {
		writeError(w, 404, "not_found", "task not found")
		return
	}
	writeJSON(w, 200, t)
}
func (a *App) deleteTask(w http.ResponseWriter, r *http.Request) {
	res, err := a.db.Exec(`DELETE FROM tasks WHERE id=?`, r.PathValue("id"))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeError(w, 404, "not_found", "task not found")
		return
	}
	w.WriteHeader(204)
}
func (a *App) findTask(id string) (Task, error) {
	var t Task
	var c string
	err := a.db.QueryRow(`SELECT id,project_id,title,description,status,created_at FROM tasks WHERE id=?`, id).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &c)
	t.CreatedAt = parseDBTime(c)
	return t, err
}
