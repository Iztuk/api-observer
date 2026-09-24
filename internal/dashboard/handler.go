// Package dashboard handles all the frontend calls
package dashboard

import (
	"api-observer/internal/audit"
	"api-observer/internal/dashboard/views/utils"
	"api-observer/internal/nodes"
	"log"
	"net/http"
)

type Handler struct {
	RuleSet *audit.RuleSet
	Nodes   *nodes.NodeManager
}

func NewHandler(rs *audit.RuleSet, nm *nodes.NodeManager) *Handler {
	return &Handler{
		RuleSet: rs,
		Nodes:   nm,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.ExplorerPage)
	mux.HandleFunc("GET /logs", h.ExplorerLogs)

	mux.HandleFunc("GET /rules", h.RulesPage)
	mux.HandleFunc("GET /rules/import", h.RulesImportPage)
	mux.HandleFunc("GET /rules/import/translate", h.RulesImportTranslate)

	mux.HandleFunc("GET /nodes", h.NodesPage)
	mux.HandleFunc("GET /nodes/list", h.NodesList)
	mux.HandleFunc("GET /nodes/delete-modal", h.DeleteNodeModal)

	mux.HandleFunc("POST /nodes", h.AddNode)
	mux.HandleFunc("POST /nodes/delete", h.DeleteNode)
}

func renderToast(
	w http.ResponseWriter,
	r *http.Request,
	httpStatus int,
	message string,
	status string,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(httpStatus)

	if err := utils.Toast(message, status).Render(r.Context(), w); err != nil {
		log.Printf("failed to render toast: %v", err)
	}
}
