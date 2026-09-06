package main

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	"api-observer-security-test/internal/app"
	"api-observer-security-test/internal/config"
	"api-observer-security-test/internal/database"

	"github.com/Iztuk/go-observer-sdk/observer"
)

func main() {
	mux := http.NewServeMux()

	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	application := app.New(mux, db)

	application.Handler()

	observerURL, err := url.Parse("http://localhost:24899")
	if err != nil {
		log.Fatalf(err.Error())
	}

	obsConf := observer.APIObserverConfig{
		Handler:      application.Mux,
		ObserverAddr: *observerURL,
		HostName:     "security-test-service",
		OpenAPI:      "./api/openapi.yaml",
		HostRules:    "./custom_rules.yaml",
	}

	handler, err := observer.Middleware(obsConf)
	if err != nil {
		log.Fatalf(err.Error())
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("API Observer Security Test Service listening on %s", cfg.Addr)
	log.Printf("SQLite database: %s", cfg.DBPath)
	log.Printf("seed users: alice@example.com/password123, bob@example.com/password123, admin@example.com/adminpass123")

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
