package dashboard

import (
	"fmt"
	"net/http"
	"observer/internal/audit"
	"observer/internal/dashboard/views"
	"strconv"
	"time"
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
	mux.HandleFunc("GET /", h.OverviewPage)
	mux.HandleFunc("GET /log-explorer", h.LogExplorerPage)
	mux.HandleFunc("GET /logs", h.GetLogExplorerLogs)

	mux.HandleFunc("GET /findings", h.FindingsPage)
}

func (h *Handler) OverviewPage(w http.ResponseWriter, r *http.Request) {

	if err := views.OverviewPage("API Observer", h.Registry.RegisteredHosts()).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render overview page.",
			http.StatusInternalServerError,
		)
	}
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

func (h *Handler) FindingsPage(w http.ResponseWriter, r *http.Request) {
	if err := views.FindingsPage("Findings").Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render findings page.",
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

	time.Sleep(2 * time.Second)

	if err := views.LogExplorerTableRows(logs.Items, logs.Cursor, limit).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render logs.",
			http.StatusInternalServerError,
		)
	}
}
