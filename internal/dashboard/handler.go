package dashboard

import (
	"api-observer/internal/dashboard/views/utils"
	"log"
	"net/http"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.LogExplorerPage)

	mux.HandleFunc("/rules", h.RulesPage)
	mux.HandleFunc("/rules/import", h.RulesImportPage)
	mux.HandleFunc("/rules/import/translate", h.RulesImportTranslate)
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
