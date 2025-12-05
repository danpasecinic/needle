package needle

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/danpasecinic/needle/internal/container"
)

type Container struct {
	internal *container.Container
	config   *containerConfig
}

type containerConfig struct {
	logger    *slog.Logger
	onResolve []ResolveHook
	onProvide []ProvideHook
	onStart   []StartHook
	onStop    []StopHook
}

func newContainer(opts ...Option) *Container {
	cfg := &containerConfig{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	internalCfg := &container.Config{
		Logger: cfg.logger,
	}

	for _, h := range cfg.onResolve {
		hook := h
		internalCfg.OnResolve = append(internalCfg.OnResolve, container.ResolveHook(hook))
	}
	for _, h := range cfg.onProvide {
		hook := h
		internalCfg.OnProvide = append(internalCfg.OnProvide, container.ProvideHook(hook))
	}
	for _, h := range cfg.onStart {
		hook := h
		internalCfg.OnStart = append(internalCfg.OnStart, container.StartHook(hook))
	}
	for _, h := range cfg.onStop {
		hook := h
		internalCfg.OnStop = append(internalCfg.OnStop, container.StopHook(hook))
	}

	return &Container{
		internal: container.New(internalCfg),
		config:   cfg,
	}
}

func (c *Container) Validate() error {
	if err := c.internal.Validate(); err != nil {
		return errValidationFailed(err)
	}
	return nil
}

func (c *Container) Size() int {
	return c.internal.Size()
}

func (c *Container) Keys() []string {
	return c.internal.Keys()
}

func (c *Container) Start(ctx context.Context) error {
	if err := c.internal.Start(ctx); err != nil {
		return errStartupFailed("container", err)
	}
	return nil
}

func (c *Container) Stop(ctx context.Context) error {
	if err := c.internal.Stop(ctx); err != nil {
		return errShutdownFailed("container", err)
	}
	return nil
}

func (c *Container) Run(ctx context.Context) error {
	if err := c.Start(ctx); err != nil {
		return err
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-quit:
	}

	signal.Stop(quit)
	close(quit)

	return c.Stop(context.Background())
}

func (c *Container) Internal() *container.Container {
	return c.internal
}

func errValidationFailed(cause error) *Error {
	return newError(ErrCodeValidationFailed, "container validation failed", cause)
}
