package needle

import (
	"log/slog"

	"github.com/danpasecinic/needle/internal/container"
)

type Container struct {
	internal *container.Container
	config   *containerConfig
}

type containerConfig struct {
	logger *slog.Logger
}

func newContainer(opts ...Option) *Container {
	cfg := &containerConfig{
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	internal := container.New(&container.Config{
		Logger: cfg.logger,
	})

	return &Container{
		internal: internal,
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

func errValidationFailed(cause error) *Error {
	return newError(ErrCodeValidationFailed, "container validation failed", cause)
}
