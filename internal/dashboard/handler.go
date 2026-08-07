package dashboard

import (
	"net/http"
	"observer/internal/dashboard/views"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.OverviewPage)
}

func (h *Handler) OverviewPage(w http.ResponseWriter, r *http.Request) {
	if err := views.OverviewPage("API Observer").Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render overview page.",
			http.StatusInternalServerError,
		)
	}
}
