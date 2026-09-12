package main

import (
	"context"
	"io"
	"log"
	"observer/internal/audit"
	"observer/internal/config"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	cfg := config.LoadConfigurationFile()

	appLogger, appLogFile, err := newLogger(
		cfg.AppLog,
		true,
	)
	if err != nil {
		log.Fatalf(
			"failed to configure application logger: %v",
			err,
		)
	}
	defer appLogFile.Close()

	findingsLogger, findingsLogFile, err := newLogger(
		cfg.FindingsLog,
		false,
	)
	if err != nil {
		appLogger.Fatalf(
			"failed to configure findings logger: %v",
			err,
		)
	}
	defer findingsLogFile.Close()

	log.SetOutput(appLogger.Writer())

	appLogger.Println("API Observer starting")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	queue := audit.NewQueue(cfg.QueueSize)

	wg := queue.StartWorkers(
		ctx,
		cfg.WorkerCount,
		findingsLogger,
	)

	defer func() {
		queue.Close()
		wg.Wait()

		appLogger.Println("workers stopped")
	}()

	<-ctx.Done()

	appLogger.Println("shutdown signal received")
	appLogger.Println("API Observer stopped")
}

func newLogger(
	logPath string,
	writeStdout bool,
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
		log.Ldate|log.Ltime,
	)

	return logger, file, nil
}
