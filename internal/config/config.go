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

// flags are parsed first, env vars override them
func Load() (*Config, error) {
	cfg := &Config{
		ServerAddress:        "localhost:8080",
		AccrualSystemAddress: "http://localhost:8081",
	}

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "listen address (overridden by RUN_ADDRESS)")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "database URI (overridden by DATABASE_URI)")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "accrual system address (overridden by ACCRUAL_SYSTEM_ADDRESS)")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
