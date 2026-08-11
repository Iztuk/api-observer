package dashboard

import (
	"fmt"
	"net/http"
	"observer/internal/audit"
	"observer/internal/dashboard/views"
	"strconv"
)

type Handler struct {
	Registry *audit.ContractRegistry
}

func NewHandler(registry *audit.ContractRegistry) *Handler {
	return &Handler{
		Registry: registry,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/log-explorer", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /log-explorer", h.LogExplorerPage)
	mux.HandleFunc("GET /logs", h.GetLogExplorerLogs)
	mux.HandleFunc("GET /logs/details", h.GetLogExplorerLogDetails)

	mux.HandleFunc("GET /rules", h.RulesPage)
}

func (h *Handler) LogExplorerPage(w http.ResponseWriter, r *http.Request) {
	logs, err := GetLogs(r.Context(), 0, 100)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch logs. %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.LogExplorerPage("Log Explorer", logs).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render log explorer page.",
			http.StatusInternalServerError,
		)
	}

}

func (h *Handler) GetLogExplorerLogs(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	cursor, err := strconv.Atoi(queryParams.Get("cursor"))
	if err != nil {
		cursor = 0
	}
	limit, err := strconv.Atoi(queryParams.Get("limit"))
	if err != nil {
		limit = 10
	}

	logs, err := GetLogs(r.Context(), int64(cursor), limit)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch logs. %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.LogExplorerTableRows(logs.Items, logs.Cursor, limit).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render logs.",
			http.StatusInternalServerError,
		)
	}
}

func (h *Handler) GetLogExplorerLogDetails(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	cursor, err := strconv.Atoi(queryParams.Get("cursor"))
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch log cursor. %s", err.Error()),
			http.StatusInternalServerError,
		)
	}

	item, err := GetLog(r.Context(), int64(cursor))
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch log. %s", err.Error()),
			http.StatusInternalServerError,
		)
	}

	if err := views.LogExplorerDetailSidebar(item).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render log details.",
			http.StatusInternalServerError,
		)
	}
}

func (h *Handler) RulesPage(w http.ResponseWriter, r *http.Request) {

	if err := views.RulesPage("Rules", h.Registry.RegisteredHosts()).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render rules page.",
			http.StatusInternalServerError,
		)
	}
}
