package dashboard

import (
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
