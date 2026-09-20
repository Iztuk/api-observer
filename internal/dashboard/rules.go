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
		renderToast(
			w,
			r,
			http.StatusBadRequest,
			"Invalid JSON body",
			"error",
		)
		return
	}

	switch req.Type {
	case "crs":
		// Separate the source into individual SecRule directives.
		rawRules, err := crs.SplitSecRules(req.Source)
		if err != nil {
			renderToast(
				w,
				r,
				http.StatusBadRequest,
				err.Error(),
				"error",
			)
			return
		}

		// Parse the directives into CRS rules.
		crsRules, err := crs.ParseRules(rawRules)
		if err != nil {
			renderToast(
				w,
				r,
				http.StatusBadRequest,
				err.Error(),
				"error",
			)
			return
		}

		// Translate all rules, including chains and warnings.
		results, err := crs.TranslateRules(crsRules)
		if err != nil {
			renderToast(
				w,
				r,
				http.StatusBadRequest,
				err.Error(),
				"error",
			)
			return
		}

		// Serialize the translation results.
		data, err := yaml.Marshal(results)
		if err != nil {
			renderToast(
				w,
				r,
				http.StatusInternalServerError,
				"Failed to serialize translation results",
				"error",
			)
			return
		}

		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(data)
		return

	default:
		renderToast(
			w,
			r,
			http.StatusBadRequest,
			fmt.Sprintf("Unsupported import type: %s", req.Type),
			"error",
		)
		return
	}
}
