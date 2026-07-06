package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"observer/internal/audit"
	"os"
	"sync"
	"time"
)

type ClientRequest struct {
	HostName  string              `json:"host"`
	OpenAPI   *audit.OpenAPIDoc   `json:"openapi,omitempty"`
	HostRules *audit.HostRulesDoc `json:"rules,omitempty"`
}

type ServerState struct {
	mu      sync.RWMutex
	clients map[string]RegisteredClient
}

type RegisteredClient struct {
	OpenAPI   *audit.OpenAPIDoc
	HostRules *audit.HostRulesDoc
}

type HTTPExchangeEvent struct {
	HostName string        `json:"host"`
	Request  *RequestCopy  `json:"request,omitempty"`
	Response *ResponseCopy `json:"response,omitempty"`
	Failure  *FailureCopy  `json:"failure,omitempty"`
}

type RequestCopy struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header"`
	Body   []byte      `json:"body"`
}

type ResponseCopy struct {
	Request    *RequestCopy `json:"request"`
	StatusCode int          `json:"status_code"`
	Headers    http.Header  `json:"headers"`
	Body       []byte       `json:"body"`
}

type FailureCopy struct {
	Request *RequestCopy `json:"request"`
	Error   string       `json:"error"`
}

func main() {
	mux := http.NewServeMux()

	serverState := &ServerState{
		clients: make(map[string]RegisteredClient),
	}

	server := &http.Server{
		Addr:         observerAddress(),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	mux.HandleFunc("POST /register-client", func(w http.ResponseWriter, r *http.Request) {
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

		_, ok := serverState.GetClient(client.HostName)
		if !ok {
			serverState.RegisterClient(client)
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read observer event", http.StatusBadRequest)
			return
		}

		var event HTTPExchangeEvent
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "invalid observer event JSON", http.StatusBadRequest)
			return
		}

		_, ok := serverState.GetClient(event.HostName)
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		pretty, err := json.MarshalIndent(event, "", "    ")
		if err != nil {
			http.Error(w, "failed to marshal json", http.StatusInternalServerError)
		}

		fmt.Println(pretty)

		w.WriteHeader(http.StatusAccepted)
	})

	fmt.Printf("Server is running on http://localhost%s\n", observerAddress())
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (s *ServerState) RegisterClient(client ClientRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[client.HostName] = RegisteredClient{
		OpenAPI:   client.OpenAPI,
		HostRules: client.HostRules,
	}

	log.Printf("New client registered: %s\n", client.HostName)
}

func (s *ServerState) GetClient(clientName string) (RegisteredClient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.clients[clientName]
	return val, ok
}

func observerAddress() string {
	if addr := os.Getenv("API_OBSERVER_ADDR"); addr != "" {
		return addr
	}

	return ":24899"
}
