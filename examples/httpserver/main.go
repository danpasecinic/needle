package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/danpasecinic/needle"
)

type Config struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func NewServer(cfg *Config, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Start(_ context.Context) error {
	s.logger.Info("starting server", "addr", s.server.Addr)
	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Error("server error", "error", err)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping server")
	return s.server.Shutdown(ctx)
}

func (s *Server) HealthCheck(_ context.Context) error {
	return nil
}

type Handler struct {
	logger *slog.Logger
}

func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("request", "method", r.Method, "path", r.URL.Path)

	switch r.URL.Path {
	case "/":
		_, _ = w.Write([]byte("Hello, World!"))
	case "/health":
		_, _ = w.Write([]byte("OK"))
	default:
		http.NotFound(w, r)
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	c := needle.New(needle.WithLogger(logger))

	_ = needle.ProvideValue(
		c, &Config{
			Port:         8080,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	)

	_ = needle.ProvideValue(c, logger)

	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (http.Handler, error) {
			log := needle.MustInvoke[*slog.Logger](c)
			return NewHandler(log), nil
		},
	)

	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
			cfg := needle.MustInvoke[*Config](c)
			handler := needle.MustInvoke[http.Handler](c)
			log := needle.MustInvoke[*slog.Logger](c)
			return NewServer(cfg, handler, log), nil
		},
		needle.WithOnStart(
			func(ctx context.Context) error {
				srv := needle.MustInvoke[*Server](c)
				return srv.Start(ctx)
			},
		),
		needle.WithOnStop(
			func(ctx context.Context) error {
				srv := needle.MustInvoke[*Server](c)
				return srv.Stop(ctx)
			},
		),
	)

	logger.Info("starting application")
	if err := c.Run(context.Background()); err != nil {
		logger.Error("application error", "error", err)
		os.Exit(1)
	}
}
