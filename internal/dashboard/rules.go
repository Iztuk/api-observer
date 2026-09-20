package dashboard

import (
	"api-observer/internal/crs"
	"api-observer/internal/dashboard/views/rules"
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

func (h *Handler) RulesPage(w http.ResponseWriter, r *http.Request) {
	if err := rules.RulesPage("Rule Page").Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) RulesImportPage(w http.ResponseWriter, r *http.Request) {
	if err := rules.RulesImportPage("Import Rule").Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type TranslateImportRequest struct {
	Source string `json:"source"`
	Type   string `json:"type"`
}

func (h *Handler) RulesImportTranslate(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req TranslateImportRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	switch req.Type {
	case "crs":
		// Separate the source into individual SecRule directives.
		rawRules, err := crs.SplitSecRules(req.Source)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse the directives into CRS rules.
		crsRules, err := crs.ParseRules(rawRules)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Translate the entire set, including chained rules and warnings.
		results, err := crs.TranslateRules(crsRules)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Convert translation results into YAML.
		data, err := yaml.Marshal(results)
		if err != nil {
			http.Error(
				w,
				"failed to serialize translation results",
				http.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(data)
		return

	default:
		http.Error(
			w,
			fmt.Sprintf("unsupported import type: %s", req.Type),
			http.StatusBadRequest,
		)
		return
	}
}
