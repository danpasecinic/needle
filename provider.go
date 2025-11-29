package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
	"github.com/danpasecinic/needle/internal/scope"
)

type Provider[T any] func(ctx context.Context, r Resolver) (T, error)

type ProviderOption func(*providerConfig)

type providerConfig struct {
	name         string
	dependencies []string
	onStart      []container.Hook
	onStop       []container.Hook
	scope        scope.Scope
	poolSize     int
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

	if err := c.internal.Register(key, wrappedProvider, cfg.dependencies); err != nil {
		return err
	}

	for _, hook := range cfg.onStart {
		c.internal.AddOnStart(key, hook)
	}
	for _, hook := range cfg.onStop {
		c.internal.AddOnStop(key, hook)
	}

	if cfg.scope != scope.Singleton {
		c.internal.SetScope(key, cfg.scope)
	}
	if cfg.poolSize > 0 {
		c.internal.SetPoolSize(key, cfg.poolSize)
	}

	return nil
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

	if err := c.internal.RegisterValue(key, value); err != nil {
		return err
	}

	for _, hook := range cfg.onStart {
		c.internal.AddOnStart(key, hook)
	}
	for _, hook := range cfg.onStop {
		c.internal.AddOnStop(key, hook)
	}

	return nil
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

func WithOnStart(hook Hook) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.onStart = append(cfg.onStart, container.Hook(hook))
	}
}

func WithOnStop(hook Hook) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.onStop = append(cfg.onStop, container.Hook(hook))
	}
}

func WithScope(s Scope) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.scope = s
	}
}

func WithPoolSize(size int) ProviderOption {
	return func(cfg *providerConfig) {
		cfg.scope = scope.Pooled
		cfg.poolSize = size
	}
}
