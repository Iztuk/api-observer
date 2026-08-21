package dashboard

import (
	"errors"
	"fmt"
	"net/http"
	"observer/internal/audit"
	"observer/internal/dashboard/views"
	"observer/internal/query"
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
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/log-explorer", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /log-explorer", h.LogExplorerPage)
	mux.HandleFunc("GET /logs", h.GetLogExplorerLogs)
	mux.HandleFunc("GET /logs/details", h.GetLogExplorerLogDetails)

	// mux.HandleFunc("GET /rules", h.RulesPage)

	mux.HandleFunc("GET /analysis", h.AnalysisPage)
	mux.HandleFunc("POST /analysis/run", h.AnalysisRun)
	mux.HandleFunc("POST /analysis/logs", h.GetAnalysisLogs)
	mux.HandleFunc("POST /analysis/logs/details", h.GetAnalysisLogDetails)
}

func (h *Handler) LogExplorerPage(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	queryString := queryParams.Get("query")

	if err := views.LogExplorerPage("Log Explorer", queryString).Render(r.Context(), w); err != nil {
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
		limit = 25
	}

	queryString := queryParams.Get("query")

	var queryErr query.QueryError

	logs, err := GetLogs(
		r.Context(),
		queryString,
		int64(cursor),
		limit,
	)

	if err != nil {
		if errors.As(err, &queryErr) {
			if renderErr := views.QueryErrorMessage(
				&queryErr,
			).Render(r.Context(), w); renderErr != nil {
				http.Error(
					w,
					"Unable to render query error.",
					http.StatusInternalServerError,
				)
			}

			return
		}

		http.Error(
			w,
			"Unable to fetch logs.",
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.LogExplorerTableRows(logs.Items, queryString, logs.Cursor, limit).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render logs.",
			http.StatusInternalServerError,
		)
		return
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
		return
	}

	item, err := GetLog(r.Context(), int64(cursor))
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch log. %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.LogExplorerDetailSidebar(item).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render log details.",
			http.StatusInternalServerError,
		)
		return
	}
}

// Disabling for now until retry mechanism is established for hosts
// func (h *Handler) RulesPage(w http.ResponseWriter, r *http.Request) {
//
// 	if err := views.RulesPage("Rules", h.Registry.RegisteredHosts()).Render(r.Context(), w); err != nil {
// 		http.Error(
// 			w,
// 			"Unable to render rules page.",
// 			http.StatusInternalServerError,
// 		)
// 	}
// }

func (h *Handler) AnalysisPage(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	today := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)

	timeFrom := today.
		AddDate(0, 0, -1).
		Format("2006-01-02T15:04")

	if err := views.AnalysisPage("Analysis", "", "", timeFrom, "").Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render analysis page.",
			http.StatusInternalServerError,
		)
	}
}

// This spawns the initial HTMX GET call
func (h *Handler) AnalysisRun(w http.ResponseWriter, r *http.Request) {
	queryString := r.FormValue("query")
	rules := r.FormValue("rules")
	from := r.FormValue("time_from")
	to := r.FormValue("time_to")

	var timeTo time.Time
	var err error
	if to == "" {
		timeTo = time.Now()
	} else {
		timeTo, err = ParseDateTimeInput(to)
		if err != nil {
			http.Error(
				w,
				"Invalid 'to' time.",
				http.StatusBadRequest,
			)
			return
		}
	}

	timeFrom, err := ParseDateTimeInput(from)
	if err != nil {
		http.Error(
			w,
			"Invalid 'from' time.",
			http.StatusBadRequest,
		)
		return
	}

	cursor, err := GetAnalysisCursor(
		r.Context(),
		timeTo,
	)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to locate analysis cursor. %w", err),
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.AnalysisResults(queryString, rules, timeFrom.Format(time.RFC3339Nano), timeTo.Format(time.RFC3339Nano), int(cursor)).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render analysis results.",
			http.StatusInternalServerError,
		)
		return
	}
}

func (h *Handler) GetAnalysisLogs(w http.ResponseWriter, r *http.Request) {
	rules := r.FormValue("rules")

	queryParams := r.URL.Query()

	queryString := queryParams.Get("query")

	cursorString := queryParams.Get("cursor")
	cursor, err := strconv.Atoi(cursorString)
	if err != nil {
		http.Error(
			w,
			"Invalid cursor.",
			http.StatusBadRequest,
		)
		return
	}

	to := queryParams.Get("to")
	from := queryParams.Get("from")

	var timeTo time.Time
	if to == "" {
		timeTo = time.Now()
	} else {
		timeTo, err = time.Parse(time.RFC3339Nano, to)
		if err != nil {
			http.Error(
				w,
				"Invalid 'to' time.",
				http.StatusBadRequest,
			)
			return
		}
	}

	timeFrom, err := time.Parse(time.RFC3339Nano, from)
	if err != nil {
		http.Error(
			w,
			"Invalid 'from' time.",
			http.StatusBadRequest,
		)
		return
	}

	logPage, timeFrom, err := GetAnalysisLogs(
		r.Context(),
		queryString,
		rules,
		query.LogCursor(cursor),
		timeFrom,
		timeTo,
	)
	if err != nil {
		http.Error(
			w,
			"Unable to fetch logs.",
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.AnalysisTableRows(
		logPage.Items,
		queryString,
		timeFrom.Format(time.RFC3339Nano),
		timeTo.Format(time.RFC3339Nano),
		rules,
		logPage.Cursor,
	).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render analysis table rows.",
			http.StatusInternalServerError,
		)
		return
	}
}

func (h *Handler) GetAnalysisLogDetails(w http.ResponseWriter, r *http.Request) {
	rules := r.FormValue("rules")

	queryParams := r.URL.Query()

	cursor, err := strconv.Atoi(queryParams.Get("cursor"))
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch log cursor. %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	item, err := GetAnalysisLog(r.Context(), rules, int64(cursor))
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Unable to fetch log. %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	if err := views.LogExplorerDetailSidebar(item).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			"Unable to render log details.",
			http.StatusInternalServerError,
		)
		return
	}
}
