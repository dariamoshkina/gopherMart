package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	httpServer *http.Server
}

func New(router *chi.Mux, addr string) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: router,
		},
	}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
