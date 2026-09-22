package dashboard

import (
	"api-observer/internal/dashboard/views/explorer"
	"net/http"
)

func (h *Handler) ExplorerPage(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()

	queryString := queries.Get("query")
	startString := queries.Get("start")
	endString := queries.Get("end")

	if err := explorer.ExplorerPage("Explorer", queryString, startString, endString).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
