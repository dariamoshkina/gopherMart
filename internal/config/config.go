package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress        string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	AuthSecret           string `env:"AUTH_SECRET" envDefault:"dev-secret-change-in-production"`
}

// flags have higher priority than environment variables
func Load() (*Config, error) {
	cfg := &Config{
		ServerAddress:        "localhost:8080",
		AccrualSystemAddress: "http://localhost:8081",
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "listen address (overrides RUN_ADDRESS)")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "database URI (overrides DATABASE_URI)")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address (overrides ACCRUAL_SYSTEM_ADDRESS)")
	flag.Parse()

	return cfg, nil
}
