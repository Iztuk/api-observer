package app

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type Project struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"ownerID"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Task struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectID"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
}

type AuditEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	UserID    string    `json:"userID,omitempty"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	SourceIP  string    `json:"sourceIP,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"timestamp"`
}
