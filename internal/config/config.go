// Package config handles the configuration loading and validation
package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppLog      string `yaml:"app_log"`
	FindingsLog string `yaml:"findings_log"` // WAL will share the same directory

	Addr        string `yaml:"addr"`
	QueueSize   int    `yaml:"queue_size"`
	WorkerCount int    `yaml:"worker_count"`
}

func LoadConfigurationFile() Config {
	cfgFilePath := os.Getenv("API_OBSERVER_CONFIG")

	// If the user explicitly provides a configuration file,
	// require that file to exist.
	if cfgFilePath != "" {
		cfg, err := loadConfiguration(cfgFilePath)
		if err != nil {
			log.Fatalf(
				"Could not load config file %q: %v",
				cfgFilePath,
				err,
			)
		}

		return cfg
	}

	// Otherwise use the default user configuration directory.
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf(
			"Could not find config directory: %v",
			err,
		)
	}

	appConfigDir := filepath.Join(
		cfgDir,
		"api-observer",
	)

	cfgFilePath = filepath.Join(
		appConfigDir,
		"config.yaml",
	)

	cfg, err := loadConfiguration(cfgFilePath)

	switch {
	case err == nil:
		return cfg

	case errors.Is(err, os.ErrNotExist):
		cfg = DefaultConfig()

		if err := cfg.Validate(); err != nil {
			log.Fatalf(
				"Invalid default configuration: %v",
				err,
			)
		}

		if err := writeDefaultConfiguration(
			appConfigDir,
			cfgFilePath,
			cfg,
		); err != nil {
			log.Fatalf(
				"Could not create default configuration: %v",
				err,
			)
		}

		return cfg

	default:
		log.Fatalf(
			"Could not load config file %q: %v",
			cfgFilePath,
			err,
		)
	}

	// log.Fatalf exits, but Go still requires a return.
	return Config{}
}

func DefaultConfig() Config {
	return Config{
		AppLog:      "logs/api-observer.log",
		FindingsLog: "logs/findings.jsonl",

		Addr:        ":24899",
		QueueSize:   1000,
		WorkerCount: 5,
	}
}

func loadConfiguration(path string) (Config, error) {
	var cfg Config

	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)

	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf(
			"failed to decode config file: %w",
			err,
		)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf(
			"invalid configuration: %w",
			err,
		)
	}

	return cfg, nil
}

func writeDefaultConfiguration(
	configDir string,
	configPath string,
	cfg Config,
) error {
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf(
			"failed to create config directory %q: %w",
			configDir,
			err,
		)
	}

	// O_EXCL prevents accidentally overwriting a configuration
	// that appeared between our existence check and this write.
	file, err := os.OpenFile(
		configPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to create config file %q: %w",
			configPath,
			err,
		)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf(
			"failed to encode default configuration: %w",
			err,
		)
	}

	return nil
}

func (cfg Config) Validate() error {
	if cfg.AppLog == "" {
		return fmt.Errorf("app_log cannot be empty")
	}

	if cfg.FindingsLog == "" {
		return fmt.Errorf("findings_log cannot be empty")
	}

	if cfg.Addr == "" {
		return fmt.Errorf("addr cannot be empty")
	}

	if cfg.QueueSize <= 0 {
		return fmt.Errorf(
			"queue_size must be greater than 0",
		)
	}

	if cfg.WorkerCount <= 0 {
		return fmt.Errorf(
			"worker_count must be greater than 0",
		)
	}

	return nil
}
