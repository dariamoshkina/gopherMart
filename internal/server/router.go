package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/dariamoshkina/gopherMart/internal/handler"
	"github.com/dariamoshkina/gopherMart/internal/middleware"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	ordersHandler *handler.OrdersHandler,
	balanceHandler *handler.BalanceHandler,
	authSecret string,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Compress)

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authSecret))
			r.Post("/orders", ordersHandler.Submit)
			r.Get("/orders", ordersHandler.List)
			r.Get("/balance", balanceHandler.GetBalance)
			r.Post("/balance/withdraw", balanceHandler.Withdraw)
			r.Get("/withdrawals", balanceHandler.ListWithdrawals)
		})
	})
	return r
}
