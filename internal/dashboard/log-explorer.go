package dashboard

import (
	"api-observer/internal/dashboard/views"
	"net/http"
)

func (h *Handler) LogExplorerPage(w http.ResponseWriter, r *http.Request) {
	if err := views.LogExplorerPage("Log Explorer").Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
