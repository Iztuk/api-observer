package server

import (
	"api-observer/internal/audit"
	"api-observer/internal/config"
	"api-observer/internal/ingest"
	ingestv1 "api-observer/proto/ingest/v1"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
)

func RunServer(ctx context.Context, background bool) error {
	cfg, err := config.LoadConfigurationFile()
	if err != nil {
		return fmt.Errorf(
			"failed to load configuration: %w",
			err,
		)
	}

	appLogger, appLogFile, err := newLogger(
		cfg.AppLog,
		!background,
		log.Ldate|log.Ltime,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to configure application logger: %w",
			err,
		)
	}
	defer appLogFile.Close()

	findingsLogger, findingsLogFile, err := newLogger(
		cfg.FindingsLog,
		false,
		0,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to configure findings logger: %w",
			err,
		)
	}
	defer findingsLogFile.Close()

	log.SetOutput(appLogger.Writer())

	var rs *audit.RuleSet

	if cfg.RuleSetPath == "" {
		rs = audit.NewRuleSet()
	} else {
		fc, err := loadFile(cfg.RuleSetPath)
		if err != nil {
			return fmt.Errorf("failed to load rule set: %w", err)
		}

		rs, err = audit.ParseRuleSet(fc)
		if err != nil {
			return fmt.Errorf("failed to parse rule set: %w", err)
		}
	}

	queue := audit.NewQueue(cfg.QueueSize)

	// Open the gRPC listener before starting workers.
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf(
			"failed to create gRPC listener: %w",
			err,
		)
	}

	// Initialize the gRPC server.
	grpcServer := grpc.NewServer()

	// Register the ingest service.
	ingestv1.RegisterIngestServiceServer(
		grpcServer,
		ingest.NewServer(queue),
	)

	// Start audit workers.
	wg := queue.StartWorkers(
		ctx,
		rs,
		cfg.WorkerCount,
		appLogger,
		findingsLogger,
	)

	// Clean up the queue when RunServer exits.
	defer func() {
		queue.Close()
		wg.Wait()

		appLogger.Println("workers stopped")
		appLogger.Println("API Observer stopped")
	}()

	// Run gRPC in a separate goroutine.
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- grpcServer.Serve(listener)
	}()

	appLogger.Printf(
		"API Observer started; gRPC listening on %s",
		listener.Addr(),
	)

	// Wait for either a shutdown signal or a server failure.
	select {
	case <-ctx.Done():
		appLogger.Println("shutdown signal received")

	case err := <-serverErr:
		if err != nil {
			grpcServer.Stop()
			return fmt.Errorf(
				"gRPC server failed: %w",
				err,
			)
		}
	}

	// Stop accepting new connections and allow active RPCs to finish.
	stopped := make(chan struct{})

	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// Don't allow a collector to keep shutdown hanging forever.
	select {
	case <-stopped:
		appLogger.Println("gRPC server stopped")

	case <-time.After(5 * time.Second):
		appLogger.Println(
			"gRPC shutdown timed out; forcing shutdown",
		)

		grpcServer.Stop()
		<-stopped
	}

	return nil
}

func newLogger(
	logPath string,
	writeStdout bool,
	flag int,
) (*log.Logger, *os.File, error) {
	dir := filepath.Dir(logPath)

	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, err
		}
	}

	file, err := os.OpenFile(
		logPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, nil, err
	}

	var writer io.Writer = file

	if writeStdout {
		writer = io.MultiWriter(
			file,
			os.Stdout,
		)
	}

	logger := log.New(
		writer,
		"",
		flag,
	)

	return logger, file, nil
}

func loadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
