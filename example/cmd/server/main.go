package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"api-observer-security-test/internal/app"
	"api-observer-security-test/internal/config"
	"api-observer-security-test/internal/database"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	application := app.New(db)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           application.Handler(),
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
