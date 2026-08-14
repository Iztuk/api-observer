package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	observer "github.com/Iztuk/go-observer-sdk/observer"
)

type application struct {
	logger *log.Logger

	mu       sync.RWMutex
	users    map[int]User
	projects map[int]Project
	tasks    map[int]Task

	nextUserID    int
	nextProjectID int
	nextTaskID    int
}

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OwnerID     int       `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type Task struct {
	ID          int        `json:"id"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	AssigneeID  *int       `json:"assignee_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     int    `json:"owner_id"`
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	AssigneeID  *int   `json:"assignee_id"`
	DueDate     string `json:"due_date"`
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *int    `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}

func main() {
	logger := log.New(
		os.Stdout,
		"",
		log.LstdFlags|log.LUTC,
	)

	app := &application{
		logger: logger,

		users: map[int]User{
			1: {
				ID:        1,
				Name:      "Alice Admin",
				Email:     "alice@example.com",
				Role:      "admin",
				CreatedAt: time.Now().UTC(),
			},
			2: {
				ID:        2,
				Name:      "Bob User",
				Email:     "bob@example.com",
				Role:      "user",
				CreatedAt: time.Now().UTC(),
			},
		},

		projects: map[int]Project{
			1: {
				ID:          1,
				Name:        "API Observer",
				Description: "API observability and auditing platform",
				OwnerID:     1,
				CreatedAt:   time.Now().UTC(),
			},
		},

		tasks: map[int]Task{
			1: {
				ID:         1,
				ProjectID:  1,
				Title:      "Build query language",
				Status:     "in_progress",
				Priority:   "high",
				AssigneeID: intPtr(1),
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
		},

		nextUserID:    3,
		nextProjectID: 2,
		nextTaskID:    2,
	}

	observerURL, err := url.Parse(
		"http://localhost:24899",
	)
	if err != nil {
		logger.Fatalf(
			"parse API Observer address: %v",
			err,
		)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", app.health)

	mux.HandleFunc("GET /users", app.listUsers)
	mux.HandleFunc("POST /users", app.createUser)
	mux.HandleFunc("GET /users/{id}", app.getUser)

	mux.HandleFunc("GET /projects", app.listProjects)
	mux.HandleFunc("POST /projects", app.createProject)
	mux.HandleFunc("GET /projects/{id}", app.getProject)

	mux.HandleFunc(
		"GET /projects/{projectID}/tasks",
		app.listProjectTasks,
	)
	mux.HandleFunc(
		"POST /projects/{projectID}/tasks",
		app.createTask,
	)

	mux.HandleFunc("GET /tasks/{id}", app.getTask)
	mux.HandleFunc("PATCH /tasks/{id}", app.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", app.deleteTask)

	mux.HandleFunc("GET /admin/stats", app.adminStats)

	observedHandler, err := observer.Middleware(
		observer.APIObserverConfig{
			Handler:      mux,
			ObserverAddr: *observerURL,
			HostName:     "task-service",
			OpenAPI:      "./openapi.yaml",
			HostRules:    "./custom_rules.yaml",
		},
	)
	if err != nil {
		logger.Fatalf(
			"initialize API Observer middleware: %v",
			err,
		)
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           requestLogger(logger, observedHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownComplete := make(chan struct{})

	go func() {
		defer close(shutdownComplete)

		shutdownSignal := make(chan os.Signal, 1)

		signal.Notify(
			shutdownSignal,
			syscall.SIGINT,
			syscall.SIGTERM,
		)

		<-shutdownSignal

		logger.Println("shutting down HTTP server")

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Printf(
				"graceful shutdown failed: %v",
				err,
			)
		}
	}()

	logger.Printf(
		"HTTP server listening on %s; Observer server: %s",
		server.Addr,
		observerURL.String(),
	)

	err = server.ListenAndServe()

	if err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf(
			"HTTP server failed: %v",
			err,
		)
	}

	<-shutdownComplete

	logger.Println("HTTP server stopped")
}

func (app *application) health(
	w http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(
		w,
		http.StatusOK,
		map[string]string{
			"status": "ok",
		},
	)
}

func (app *application) listUsers(
	w http.ResponseWriter,
	r *http.Request,
) {
	role := r.URL.Query().Get("role")

	app.mu.RLock()
	defer app.mu.RUnlock()

	users := make([]User, 0)

	for _, user := range app.users {
		if role != "" && user.Role != role {
			continue
		}

		users = append(users, user)
	}

	writeJSON(
		w,
		http.StatusOK,
		users,
	)
}

func (app *application) createUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input createUserRequest

	if err := decodeJSON(r, &input); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	input.Role = strings.TrimSpace(input.Role)

	if input.Name == "" {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"name is required",
		)
		return
	}

	if input.Email == "" ||
		!strings.Contains(input.Email, "@") {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"valid email is required",
		)
		return
	}

	switch input.Role {
	case "admin", "user":
	default:
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"role must be admin or user",
		)
		return
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	for _, user := range app.users {
		if strings.EqualFold(
			user.Email,
			input.Email,
		) {
			writeError(
				w,
				http.StatusConflict,
				"email already exists",
			)
			return
		}
	}

	user := User{
		ID:        app.nextUserID,
		Name:      input.Name,
		Email:     input.Email,
		Role:      input.Role,
		CreatedAt: time.Now().UTC(),
	}

	app.users[user.ID] = user
	app.nextUserID++

	writeJSON(
		w,
		http.StatusCreated,
		user,
	)
}

func (app *application) getUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parsePositiveID(
		r.PathValue("id"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid user ID",
		)
		return
	}

	app.mu.RLock()
	user, ok := app.users[id]
	app.mu.RUnlock()

	if !ok {
		writeError(
			w,
			http.StatusNotFound,
			"user not found",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		user,
	)
}

func (app *application) listProjects(
	w http.ResponseWriter,
	r *http.Request,
) {
	ownerID := 0

	if rawOwnerID := r.URL.Query().Get("owner_id"); rawOwnerID != "" {
		value, err := parsePositiveID(rawOwnerID)

		if err != nil {
			writeError(
				w,
				http.StatusBadRequest,
				"invalid owner_id",
			)
			return
		}

		ownerID = value
	}

	app.mu.RLock()
	defer app.mu.RUnlock()

	projects := make([]Project, 0)

	for _, project := range app.projects {
		if ownerID != 0 &&
			project.OwnerID != ownerID {
			continue
		}

		projects = append(
			projects,
			project,
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		projects,
	)
}

func (app *application) createProject(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input createProjectRequest

	if err := decodeJSON(r, &input); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	if strings.TrimSpace(input.Name) == "" {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"project name is required",
		)
		return
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if _, ok := app.users[input.OwnerID]; !ok {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"owner does not exist",
		)
		return
	}

	project := Project{
		ID:          app.nextProjectID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		OwnerID:     input.OwnerID,
		CreatedAt:   time.Now().UTC(),
	}

	app.projects[project.ID] = project
	app.nextProjectID++

	writeJSON(
		w,
		http.StatusCreated,
		project,
	)
}

func (app *application) getProject(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parsePositiveID(
		r.PathValue("id"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid project ID",
		)
		return
	}

	app.mu.RLock()
	project, ok := app.projects[id]
	app.mu.RUnlock()

	if !ok {
		writeError(
			w,
			http.StatusNotFound,
			"project not found",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		project,
	)
}

func (app *application) listProjectTasks(
	w http.ResponseWriter,
	r *http.Request,
) {
	projectID, err := parsePositiveID(
		r.PathValue("projectID"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid project ID",
		)
		return
	}

	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")

	app.mu.RLock()
	defer app.mu.RUnlock()

	if _, ok := app.projects[projectID]; !ok {
		writeError(
			w,
			http.StatusNotFound,
			"project not found",
		)
		return
	}

	tasks := make([]Task, 0)

	for _, task := range app.tasks {
		if task.ProjectID != projectID {
			continue
		}

		if status != "" &&
			task.Status != status {
			continue
		}

		if priority != "" &&
			task.Priority != priority {
			continue
		}

		tasks = append(tasks, task)
	}

	writeJSON(
		w,
		http.StatusOK,
		tasks,
	)
}

func (app *application) createTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	projectID, err := parsePositiveID(
		r.PathValue("projectID"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid project ID",
		)
		return
	}

	var input createTaskRequest

	if err := decodeJSON(r, &input); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	if strings.TrimSpace(input.Title) == "" {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"title is required",
		)
		return
	}

	if !validPriority(input.Priority) {
		writeError(
			w,
			http.StatusUnprocessableEntity,
			"invalid priority",
		)
		return
	}

	var dueDate *time.Time

	if input.DueDate != "" {
		parsed, err := time.Parse(
			time.RFC3339,
			input.DueDate,
		)
		if err != nil {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"invalid due_date",
			)
			return
		}

		dueDate = &parsed
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if _, ok := app.projects[projectID]; !ok {
		writeError(
			w,
			http.StatusNotFound,
			"project not found",
		)
		return
	}

	if input.AssigneeID != nil {
		if _, ok := app.users[*input.AssigneeID]; !ok {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"assignee does not exist",
			)
			return
		}
	}

	now := time.Now().UTC()

	task := Task{
		ID:          app.nextTaskID,
		ProjectID:   projectID,
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Status:      "todo",
		Priority:    input.Priority,
		AssigneeID:  input.AssigneeID,
		DueDate:     dueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	app.tasks[task.ID] = task
	app.nextTaskID++

	writeJSON(
		w,
		http.StatusCreated,
		task,
	)
}

func (app *application) getTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parsePositiveID(
		r.PathValue("id"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid task ID",
		)
		return
	}

	app.mu.RLock()
	task, ok := app.tasks[id]
	app.mu.RUnlock()

	if !ok {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		task,
	)
}

func (app *application) updateTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parsePositiveID(
		r.PathValue("id"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid task ID",
		)
		return
	}

	var input updateTaskRequest

	if err := decodeJSON(r, &input); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	task, ok := app.tasks[id]
	if !ok {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)

		if title == "" {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"title cannot be empty",
			)
			return
		}

		task.Title = title
	}

	if input.Description != nil {
		task.Description = strings.TrimSpace(
			*input.Description,
		)
	}

	if input.Status != nil {
		if !validStatus(*input.Status) {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"invalid status",
			)
			return
		}

		task.Status = *input.Status
	}

	if input.Priority != nil {
		if !validPriority(*input.Priority) {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"invalid priority",
			)
			return
		}

		task.Priority = *input.Priority
	}

	if input.AssigneeID != nil {
		if _, ok := app.users[*input.AssigneeID]; !ok {
			writeError(
				w,
				http.StatusUnprocessableEntity,
				"assignee does not exist",
			)
			return
		}

		task.AssigneeID = input.AssigneeID
	}

	if input.DueDate != nil {
		if *input.DueDate == "" {
			task.DueDate = nil
		} else {
			parsed, err := time.Parse(
				time.RFC3339,
				*input.DueDate,
			)
			if err != nil {
				writeError(
					w,
					http.StatusUnprocessableEntity,
					"invalid due_date",
				)
				return
			}

			task.DueDate = &parsed
		}
	}

	task.UpdatedAt = time.Now().UTC()

	app.tasks[id] = task

	writeJSON(
		w,
		http.StatusOK,
		task,
	)
}

func (app *application) deleteTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := parsePositiveID(
		r.PathValue("id"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid task ID",
		)
		return
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if _, ok := app.tasks[id]; !ok {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	delete(app.tasks, id)

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) adminStats(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Header.Get("X-Admin-Key") != "observer-test-admin" {
		writeError(
			w,
			http.StatusUnauthorized,
			"invalid admin credentials",
		)
		return
	}

	app.mu.RLock()
	defer app.mu.RUnlock()

	writeJSON(
		w,
		http.StatusOK,
		map[string]any{
			"users":    len(app.users),
			"projects": len(app.projects),
			"tasks":    len(app.tasks),
		},
	)
}

func decodeJSON(
	r *http.Request,
	dst any,
) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf(
			"invalid JSON request: %w",
			err,
		)
	}

	return nil
}

func parsePositiveID(
	value string,
) (int, error) {
	id, err := strconv.Atoi(value)

	if err != nil || id < 1 {
		return 0, errors.New(
			"ID must be a positive integer",
		)
	}

	return id, nil
}

func validStatus(value string) bool {
	switch value {
	case "todo",
		"in_progress",
		"done":
		return true

	default:
		return false
	}
}

func validPriority(value string) bool {
	switch value {
	case "low",
		"medium",
		"high":
		return true

	default:
		return false
	}
}

func requestLogger(
	logger *log.Logger,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			started := time.Now()

			next.ServeHTTP(w, r)

			logger.Printf(
				"%s %s duration=%s",
				r.Method,
				r.URL.RequestURI(),
				time.Since(started),
			)
		},
	)
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(
			w,
			http.StatusText(
				http.StatusInternalServerError,
			),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	if _, err := w.Write(
		append(data, '\n'),
	); err != nil {
		log.Printf(
			"write JSON response: %v",
			err,
		)
	}
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(
		w,
		status,
		map[string]string{
			"error": message,
		},
	)
}

func intPtr(value int) *int {
	return &value
}
