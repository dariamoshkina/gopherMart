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

	"go.uber.org/zap"

	"github.com/dariamoshkina/gopherMart/internal/client/accrual"
	"github.com/dariamoshkina/gopherMart/internal/config"
	"github.com/dariamoshkina/gopherMart/internal/handler"
	"github.com/dariamoshkina/gopherMart/internal/infra"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := infra.NewPgPool(ctx, cfg.DatabaseURI, logger)
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

	accrualClient := accrual.New(cfg.AccrualSystemAddress, logger)
	poller := worker.New(orderRepo, accrualClient, 2*time.Second, logger)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		poller.Run(ctx)
	}()

	logger.Info("starting server", zap.String("addr", cfg.ServerAddress))
	go func() {
		if err = srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown", zap.Error(err))
	}

	wg.Wait()
	logger.Info("goodbye")
}
