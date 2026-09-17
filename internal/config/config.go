// Package config handles the configuration loading and validation
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AppLog      string `yaml:"app_log"`
	FindingsLog string `yaml:"findings_log"` // WAL will share the same directory
	RuleSetPath string `yaml:"rules"`

	Addr        string `yaml:"addr"`
	QueueSize   int    `yaml:"queue_size"`
	WorkerCount int    `yaml:"worker_count"`
}

func LoadConfigurationFile() (Config, error) {
	cfgFilePath := os.Getenv("API_OBSERVER_CONFIG")

	if cfgFilePath == "" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf(
				"could not find config directory: %w",
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
			return cfg, nil

		case errors.Is(err, os.ErrNotExist):
			c, err := DefaultConfig()
			if err != nil {
				return Config{}, fmt.Errorf("%s", err.Error())
			}
			cfg = c

			if err := cfg.Validate(); err != nil {
				return Config{}, fmt.Errorf(
					"invalid default configuration: %w",
					err,
				)
			}

			if err := writeDefaultConfiguration(
				appConfigDir,
				cfgFilePath,
				cfg,
			); err != nil {
				return Config{}, fmt.Errorf(
					"could not create default configuration: %w",
					err,
				)
			}

			return cfg, nil

		default:
			return Config{}, fmt.Errorf(
				"could not load config file %q: %w",
				cfgFilePath,
				err,
			)
		}
	}

	cfg, err := loadConfiguration(cfgFilePath)
	if err != nil {
		return Config{}, fmt.Errorf(
			"could not load config file %q: %w",
			cfgFilePath,
			err,
		)
	}

	return cfg, nil
}

func DefaultConfig() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, fmt.Errorf(
			"failed to get user config directory: %w",
			err,
		)
	}

	appDir := filepath.Join(configDir, "api-observer")

	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return Config{}, fmt.Errorf(
			"failed to create application config directory: %w",
			err,
		)
	}

	return Config{
		AppLog: filepath.Join(
			appDir,
			"logs",
			"api-observer.log",
		),

		FindingsLog: filepath.Join(
			appDir,
			"logs",
			"findings.jsonl",
		),

		RuleSetPath: filepath.Join(
			appDir,
			"rules.yaml",
		),

		Addr:        ":24899",
		QueueSize:   1000,
		WorkerCount: 5,
	}, nil
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
