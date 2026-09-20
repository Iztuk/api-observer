package dashboard

import (
	"api-observer/internal/dashboard/views/rules"
	"fmt"
	"net/http"
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

func (h *Handler) RulesImportTranslate(
	w http.ResponseWriter,
	r *http.Request,
) {
	importValue := r.FormValue("import-editor-value")

	// Replace importValue with translated YAML later.

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	fmt.Fprintf(w, importValue)
}
