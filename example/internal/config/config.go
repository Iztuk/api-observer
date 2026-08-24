package config

import "os"

type Config struct {
	Addr   string
	DBPath string
}

func Load() Config {
	cfg := Config{
		Addr:   ":8080",
		DBPath: "./test.db",
	}

	if v := os.Getenv("TEST_SERVICE_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("TEST_SERVICE_DB"); v != "" {
		cfg.DBPath = v
	}
	return cfg
}
