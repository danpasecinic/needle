package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

func Replace[T any](c *Container, provider Provider[T], opts ...ProviderOption) error {
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

	if err := c.internal.Replace(key, wrappedProvider, cfg.dependencies); err != nil {
		return err
	}

	for _, hook := range cfg.onStart {
		c.internal.AddOnStart(key, hook)
	}
	for _, hook := range cfg.onStop {
		c.internal.AddOnStop(key, hook)
	}

	if cfg.scope != 0 {
		c.internal.SetScope(key, cfg.scope)
	}
	if cfg.poolSize > 0 {
		c.internal.SetPoolSize(key, cfg.poolSize)
	}
	if cfg.lazy {
		c.internal.SetLazy(key, true)
	}

	return nil
}

func ReplaceValue[T any](c *Container, value T, opts ...ProviderOption) error {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	key := reflect.TypeKey[T]()
	if cfg.name != "" {
		key = reflect.TypeKeyNamed[T](cfg.name)
	}

	if err := c.internal.ReplaceValue(key, value); err != nil {
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

func ReplaceNamed[T any](c *Container, name string, provider Provider[T], opts ...ProviderOption) error {
	opts = append(opts, WithName(name))
	return Replace(c, provider, opts...)
}

func ReplaceNamedValue[T any](c *Container, name string, value T, opts ...ProviderOption) error {
	opts = append(opts, WithName(name))
	return ReplaceValue(c, value, opts...)
}

func MustReplace[T any](c *Container, provider Provider[T], opts ...ProviderOption) {
	if err := Replace(c, provider, opts...); err != nil {
		panic(err)
	}
}

func MustReplaceValue[T any](c *Container, value T, opts ...ProviderOption) {
	if err := ReplaceValue(c, value, opts...); err != nil {
		panic(err)
	}
}
