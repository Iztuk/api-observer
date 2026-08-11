package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"observer/internal/audit"
	"observer/internal/dashboard"
	"observer/internal/ingest"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	logger := log.New(os.Stdout, "api-observer: ", log.LstdFlags|log.Lmicroseconds|log.LUTC)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	registry := audit.NewContractRegistry()

	engine := audit.NewRuleEngine(registry)

	queue := audit.NewQueue(observerQueueSize())
	store, err := audit.NewJSONLogStore()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer store.Close()

	wg := queue.StartWorkers(ctx, observerWorkerCount(), logger, engine, store)

	ingestHandler := ingest.NewHandler(registry, queue, engine)

	ingestHandler.RegisterRoutes(mux)

	defer func() {
		queue.Close()
		wg.Wait()
	}()

	server := &http.Server{
		Addr:         observerAddress(),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	fileServer := http.FileServer(
		http.Dir("./internal/dashboard/views/assets"),
	)

	mux.Handle(
		"GET /static/",
		http.StripPrefix("/static/", fileServer),
	)

	// Dashboard
	dashboardHandler := dashboard.NewHandler(registry)
	dashboardHandler.RegisterRoutes(mux)

	fmt.Printf("Server is running on http://localhost%s\n", observerAddress())

	serverErr := make(chan error, 1)

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("server shutdown failed: %v", err)
		}

		logger.Println("server stopped")

	case err := <-serverErr:
		if err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}
}

func observerAddress() string {
	if addr := os.Getenv("API_OBSERVER_ADDR"); addr != "" {
		return addr
	}

	return ":24899"
}

func observerQueueSize() int {
	if size := os.Getenv("API_OBSERVER_QUEUE_SIZE"); size != "" {
		num, err := strconv.Atoi(size)
		if err != nil {
			return 1000
		}

		return num
	}

	return 1000
}

func observerWorkerCount() int {
	if size := os.Getenv("API_OBSERVER_WORKER_COUNT"); size != "" {
		num, err := strconv.Atoi(size)
		if err != nil {
			return 5
		}

		return num
	}

	return 5

}
