package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"observer/internal/audit"
	"os"
	"strings"
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

		err = ProcessHTTPEvent(r.Context(), event)
		if err != nil {
			http.Error(w, "could not process event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	})

	fmt.Printf("Server is running on http://localhost%s\n", observerAddress())
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func ProcessHTTPEvent(ctx context.Context, event HTTPExchangeEvent) error {
	if strings.TrimSpace(event.HostName) == "" {
		return fmt.Errorf("http exchange event missing host")
	}

	eventCount := 0

	if event.Request != nil {
		eventCount++
	}

	if event.Response != nil {
		eventCount++
	}

	if event.Failure != nil {
		eventCount++
	}

	if eventCount == 0 {
		return fmt.Errorf("http exchange event has no request, response, or failure payload")
	}

	if eventCount > 1 {
		return fmt.Errorf("http exchange event must contain only one payload type")
	}

	switch {
	case event.Request != nil:
		return processRequestEvent(ctx, event.HostName, event.Request)

	case event.Response != nil:
		return processResponseEvent(ctx, event.HostName, event.Response)

	case event.Failure != nil:
		return processFailureEvent(ctx, event.HostName, event.Failure)

	default:
		return fmt.Errorf("unsupported http exchange event")
	}
}

func processRequestEvent(ctx context.Context, host string, reqCopy *RequestCopy) error {
	if reqCopy == nil {
		return fmt.Errorf("request event missing request")
	}

	req, err := http.NewRequest(
		reqCopy.Method,
		reqCopy.URL,
		bytes.NewReader(reqCopy.Body),
	)
	if err != nil {
		return err
	}

	req.Header = reqCopy.Header.Clone()
	req.ContentLength = int64(len(reqCopy.Body))

	job := audit.NewRequestJob(req, host, time.Now().UTC())

	job.Body = reqCopy.Body

	data, err := json.MarshalIndent(job, "", "    ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))

	_ = ctx
	return nil
}

func processResponseEvent(ctx context.Context, host string, resCopy *ResponseCopy) error {

	return nil
}

func processFailureEvent(ctx context.Context, host string, failCopy *FailureCopy) error {

	return nil
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
