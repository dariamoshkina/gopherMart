package infra

import (
	"context"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPgPool(ctx context.Context, databaseURI string, logger *zap.Logger) (*pgxpool.Pool, error) {
	if err := RunPgMigrations(databaseURI); err != nil {
		return nil, err
	}
	logger.Info("migrations applied")
	config, err := pgxpool.ParseConfig(databaseURI)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func RunPgMigrations(databaseURI string) error {
	m, err := migrate.New("file://migrations", databaseURI)
	if err != nil {
		return err
	}
	defer m.Close()
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
