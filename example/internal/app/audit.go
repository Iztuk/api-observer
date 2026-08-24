package app

import (
	"net"
	"net/http"
	"time"
)

func (a *App) audit(r *http.Request, eventType, userID string, status int, message string) {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host == "" {
		host = r.RemoteAddr
	}
	_, _ = a.db.Exec(`INSERT INTO audit_events(id,event_type,user_id,method,path,status,source_ip,message,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		newID(), eventType, userID, r.Method, r.URL.Path, status, host, message, time.Now().UTC().Format(time.RFC3339Nano))
}

func (a *App) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.requireAdmin(w, r); !ok {
		return
	}
	// Intentionally incomplete application logging (A09): many ordinary reads,
	// debug access, and search requests are not inserted into audit_events.
	rows, err := a.db.Query(`SELECT id,event_type,COALESCE(user_id,''),method,path,status,source_ip,message,created_at FROM audit_events ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	defer rows.Close()
	events := []AuditEvent{}
	for rows.Next() {
		var e AuditEvent
		var c string
		if rows.Scan(&e.ID, &e.Type, &e.UserID, &e.Method, &e.Path, &e.Status, &e.SourceIP, &e.Message, &c) == nil {
			e.CreatedAt = parseDBTime(c)
			events = append(events, e)
		}
	}
	writeJSON(w, 200, events)
}
