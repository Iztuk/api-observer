package dashboard

import (
	"api-observer/internal/audit"
	"api-observer/internal/dashboard/views/utils"
	"log"
	"net/http"
)

type Handler struct {
	RuleSet *audit.RuleSet
}

func NewHandler(rs *audit.RuleSet) *Handler {
	return &Handler{
		RuleSet: rs,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.ExplorerPage)

	mux.HandleFunc("GET /rules", h.RulesPage)
	mux.HandleFunc("GET /rules/import", h.RulesImportPage)
	mux.HandleFunc("GET /rules/import/translate", h.RulesImportTranslate)
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
