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

		var event any
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "invalid observer event JSON", http.StatusBadRequest)
			return
		}

		pretty, err := json.MarshalIndent(event, "", "    ")
		if err != nil {
			http.Error(w, "failed to pretty print observer event", http.StatusInternalServerError)
			return
		}

		fmt.Println("---- OBSERVER EVENT RECEIVED ----")
		fmt.Println(string(pretty))

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
