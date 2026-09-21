package dashboard

import (
	"api-observer/internal/audit"
	"api-observer/internal/crs"
	"api-observer/internal/dashboard/views/rules"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
)

func (h *Handler) RulesPage(w http.ResponseWriter, r *http.Request) {
	ruleItems := ruleListItems(h.RuleSet)

	if err := rules.RulesPage("Rule Page", ruleItems).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ruleListItems(rs *audit.RuleSet) []rules.RuleListItem {
	items := make([]rules.RuleListItem, 0)

	if rs == nil {
		return items
	}

	for id, rule := range rs.Rules {
		if rule == nil {
			continue
		}

		items = append(items, rules.RuleListItem{
			ID:   id,
			Rule: rule,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items
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
		data, err := results.RuleSetYAML()
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
