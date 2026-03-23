package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"

	"github.com/dariamoshkina/gopherMart/internal/client/accrual"
	"github.com/dariamoshkina/gopherMart/internal/config"
	"github.com/dariamoshkina/gopherMart/internal/handler"
	"github.com/dariamoshkina/gopherMart/internal/repository/postgres"
	"github.com/dariamoshkina/gopherMart/internal/server"
	"github.com/dariamoshkina/gopherMart/internal/service"
	"github.com/dariamoshkina/gopherMart/internal/worker"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to create logger: " + err.Error())
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	if err := runMigrations(cfg.DatabaseURI); err != nil {
		logger.Fatal("migrations", zap.Error(err))
	}
	logger.Info("migrations applied")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURI)
	if err != nil {
		logger.Fatal("database", zap.Error(err))
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	orderRepo := postgres.NewOrderRepository(pool)
	balanceRepo := postgres.NewBalanceRepository(pool)

	authSvc := service.NewAuthService(userRepo)
	ordersSvc := service.NewOrdersService(orderRepo)
	balanceSvc := service.NewBalanceService(balanceRepo)

	authHandler := handler.NewAuthHandler(authSvc, cfg.AuthSecret)
	ordersHandler := handler.NewOrdersHandler(ordersSvc)
	balanceHandler := handler.NewBalanceHandler(balanceSvc)

	router := server.NewRouter(authHandler, ordersHandler, balanceHandler, cfg.AuthSecret)
	srv := server.New(router, cfg.ServerAddress)

	accrualClient := accrual.New(cfg.AccrualSystemAddress)
	poller := worker.New(orderRepo, accrualClient, 2*time.Second, logger)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		poller.Run(ctx)
	}()

	logger.Info("starting server", zap.String("addr", cfg.ServerAddress))
	go func() {
		if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown", zap.Error(err))
	}

	wg.Wait()
	logger.Info("goodbye")
}

func runMigrations(databaseURL string) error {
	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
