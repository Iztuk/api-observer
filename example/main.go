package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	observer "github.com/Iztuk/go-observer-sdk/observer"
)

type application struct {
	logger *log.Logger
}

type echoRequest struct {
	Message string `json:"message"`
}

type echoResponse struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.LUTC)

	app := &application{
		logger: logger,
	}

	observerURL, err := url.Parse("http://localhost:24899")
	if err != nil {
		logger.Fatalf("parse API_OBSERVER_ADDR: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", app.health)
	mux.HandleFunc("GET /hello", app.hello)
	mux.HandleFunc("POST /echo", app.echo)
	mux.HandleFunc("GET /users/{id}", app.getUser)
	mux.HandleFunc("GET /fail", app.fail)

	observedHandler, err := observer.Middleware(observer.APIObserverConfig{
		Handler:      mux,
		ObserverAddr: *observerURL,
		HostName:     "test-service",
		OpenAPI:      "./openapi.yaml",
		HostRules:    "./custom_rules.yaml",
	})
	if err != nil {
		logger.Fatalf("initialize API Observer middleware: %v", err)
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           requestLogger(logger, observedHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownComplete := make(chan struct{})

	go func() {
		defer close(shutdownComplete)

		shutdownSignal := make(chan os.Signal, 1)
		signal.Notify(
			shutdownSignal,
			syscall.SIGINT,
			syscall.SIGTERM,
		)

		<-shutdownSignal

		logger.Println("shutting down HTTP server")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Printf("graceful shutdown failed: %v", err)
		}
	}()

	logger.Printf(
		"HTTP server listening on %s; Observer server: %s",
		server.Addr,
		observerURL.String(),
	)

	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("HTTP server failed: %v", err)
	}

	<-shutdownComplete
	logger.Println("HTTP server stopped")
}

func (app *application) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (app *application) hello(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Hello from the observed API",
	})
}

func (app *application) echo(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var input echoRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	if input.Message == "" {
		writeError(w, http.StatusUnprocessableEntity, "message is required")
		return
	}

	writeJSON(w, http.StatusOK, echoResponse{
		Message:   input.Message,
		Timestamp: time.Now().UTC(),
	})
}

func (app *application) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"name":  fmt.Sprintf("User %d", id),
		"email": fmt.Sprintf("user%d@example.com", id),
	})
}

func (app *application) fail(w http.ResponseWriter, _ *http.Request) {
	writeError(
		w,
		http.StatusInternalServerError,
		"intentional test failure",
	)
}

func requestLogger(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()

		next.ServeHTTP(w, r)

		logger.Printf(
			"%s %s duration=%s",
			r.Method,
			r.URL.RequestURI(),
			time.Since(started),
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(append(data, '\n')); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
