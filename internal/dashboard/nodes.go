package dashboard

import (
	nodespage "api-observer/internal/dashboard/views/nodes-page"
	"api-observer/internal/nodes"
	"fmt"
	"net/http"
	"slices"
)

func (h *Handler) NodesPage(w http.ResponseWriter, r *http.Request) {
	if err := nodespage.NodesPage(
		"Nodes",
		h.Nodes.List(),
	).Render(r.Context(), w); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
}

func (h *Handler) NodesList(w http.ResponseWriter, r *http.Request) {
	if err := nodespage.NodesWorkspace(
		h.Nodes.List(),
	).Render(r.Context(), w); err != nil {
		renderToast(
			w,
			r,
			http.StatusInternalServerError,
			"failed to render node list",
			"error",
		)
		return
	}
}

func (h *Handler) DeleteNodeModal(w http.ResponseWriter, r *http.Request) {
	if err := nodespage.DeleteNodeModal(
		h.Nodes.List(),
	).Render(r.Context(), w); err != nil {
		renderToast(
			w,
			r,
			http.StatusInternalServerError,
			"failed to render delete node modal",
			"error",
		)
		return
	}
}

func (h *Handler) AddNode(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	addr := r.FormValue("addr")

	found := slices.ContainsFunc(
		h.Nodes.List(),
		func(node *nodes.Node) bool {
			return node.Name == name
		},
	)

	if found {
		renderToast(
			w,
			r,
			http.StatusConflict,
			fmt.Sprintf("node '%s' already exists", name),
			"warning",
		)
		return
	}

	if err := h.Nodes.Add(name, addr); err != nil {
		renderToast(
			w,
			r,
			http.StatusInternalServerError,
			"failed to add node",
			"error",
		)
		return
	}

	w.Header().Set("HX-Trigger", "nodesChanged")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("id")

	found := slices.ContainsFunc(
		h.Nodes.List(),
		func(node *nodes.Node) bool {
			return node.Name == name
		},
	)

	if !found {
		renderToast(
			w,
			r,
			http.StatusNotFound,
			fmt.Sprintf("node '%s' not found", name),
			"warning",
		)
		return
	}

	if err := h.Nodes.Remove(name); err != nil {
		renderToast(
			w,
			r,
			http.StatusInternalServerError,
			fmt.Sprintf(
				"failed to remove node '%s'",
				name,
			),
			"error",
		)
		return
	}

	w.Header().Set("HX-Trigger", "nodesChanged")
	w.WriteHeader(http.StatusNoContent)
}
