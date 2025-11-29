package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type Provider[T any] func(ctx context.Context, r Resolver) (T, error)

type ProviderOption func(*providerConfig)

type providerConfig struct {
	name         string
	dependencies []string
}

func Provide[T any](c *Container, provider Provider[T], opts ...ProviderOption) error {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	key := reflect.TypeKey[T]()
	if cfg.name != "" {
		key = reflect.TypeKeyNamed[T](cfg.name)
	}

	wrappedProvider := func(ctx context.Context, r container.Resolver) (any, error) {
		resolver := &resolverAdapter{container: c}
		return provider(ctx, resolver)
	}

	return c.internal.Register(key, wrappedProvider, cfg.dependencies)
}

func ProvideValue[T any](c *Container, value T, opts ...ProviderOption) error {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	key := reflect.TypeKey[T]()
	if cfg.name != "" {
		key = reflect.TypeKeyNamed[T](cfg.name)
	}

	return c.internal.RegisterValue(key, value)
}

func ProvideNamed[T any](c *Container, name string, provider Provider[T], opts ...ProviderOption) error {
	opts = append(opts, WithName(name))
	return Provide(c, provider, opts...)
}

func ProvideNamedValue[T any](c *Container, name string, value T, opts ...ProviderOption) error {
	opts = append(opts, WithName(name))
	return ProvideValue(c, value, opts...)
}

func WithName(name string) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.name = name
	}
}

func WithDependencies(deps ...string) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.dependencies = deps
	}
}
