package ingest

import (
	"encoding/json"
	"io"
	"net/http"
	"observer/internal/audit"
)

type ClientRequest struct {
	HostName  string              `json:"host"`
	OpenAPI   *audit.OpenAPIDoc   `json:"openapi,omitempty"`
	HostRules *audit.HostRulesDoc `json:"rules,omitempty"`
}

type Handler struct {
	Registry   *audit.ContractRegistry
	Queue      *audit.Queue
	RuleEngine *audit.RuleEngine
}

func NewHandler(registry *audit.ContractRegistry, queue *audit.Queue, engine *audit.RuleEngine) *Handler {
	return &Handler{
		Registry:   registry,
		Queue:      queue,
		RuleEngine: engine,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /register-client", h.RegisterClient)
	mux.HandleFunc("POST /events", h.Events)

}

func (h *Handler) RegisterClient(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read client registration body", http.StatusBadRequest)
		return
	}

	var client ClientRequest
	if err := json.Unmarshal(body, &client); err != nil {
		http.Error(w, "invalid client registration JSON", http.StatusBadRequest)
		return
	}

	ok := h.Registry.HostExists(client.HostName)
	if !ok {
		h.Registry.RegisterHost(client.HostName, *client.OpenAPI, *client.HostRules)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read observer event", http.StatusBadRequest)
		return
	}

	var event audit.HTTPExchangeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid observer event JSON", http.StatusBadRequest)
		return
	}

	if ok := h.Registry.HostExists(event.HostName); !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = h.Queue.ProcessHTTPEvent(r.Context(), event, *h.RuleEngine)
	if err != nil {
		http.Error(w, "could not process event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
