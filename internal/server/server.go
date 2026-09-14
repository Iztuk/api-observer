package server

import (
	"api-observer/internal/audit"
	"api-observer/internal/config"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

	fc, err := loadFile(cfg.RuleSetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fc = ""
		} else {
			return fmt.Errorf(
				"failed to load rule set: %w",
				err,
			)
		}
	}

	rs, err := audit.ParseRuleSet(fc)
	if err != nil {
		return fmt.Errorf(
			"failed to parse rule set: %w",
			err,
		)
	}

	queue := audit.NewQueue(cfg.QueueSize)

	wg := queue.StartWorkers(
		ctx,
		rs,
		cfg.WorkerCount,
		appLogger,
		findingsLogger,
	)

	defer func() {
		queue.Close()
		wg.Wait()

		appLogger.Println("workers stopped")
		appLogger.Println("API Observer stopped")
	}()

	<-ctx.Done()

	appLogger.Println("shutdown signal received")

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
