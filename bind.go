package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type Decorator[T any] func(ctx context.Context, r Resolver, base T) (T, error)

func Bind[I, T any](c *Container, opts ...ProviderOption) error {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	interfaceKey := reflect.TypeKey[I]()
	implKey := reflect.TypeKey[T]()

	if cfg.name != "" {
		interfaceKey = reflect.TypeKeyNamed[I](cfg.name)
	}

	wrappedProvider := func(ctx context.Context, r container.Resolver) (any, error) {
		return r.Resolve(ctx, implKey)
	}

	if err := c.internal.Register(interfaceKey, wrappedProvider, []string{implKey}); err != nil {
		return err
	}

	for _, hook := range cfg.onStart {
		c.internal.AddOnStart(interfaceKey, hook)
	}
	for _, hook := range cfg.onStop {
		c.internal.AddOnStop(interfaceKey, hook)
	}

	return nil
}

func BindNamed[I, T any](c *Container, name string, opts ...ProviderOption) error {
	opts = append(opts, WithName(name))
	return Bind[I, T](c, opts...)
}

func Decorate[T any](c *Container, decorator Decorator[T]) {
	key := reflect.TypeKey[T]()

	c.internal.AddDecorator(
		key, func(ctx context.Context, r container.Resolver, instance any) (any, error) {
			typed, ok := instance.(T)
			if !ok {
				var zero T
				return zero, errDecoratorTypeMismatch(reflect.TypeName[T]())
			}

			resolver := &resolverAdapter{container: c}
			return decorator(ctx, resolver, typed)
		},
	)
}

func DecorateNamed[T any](c *Container, name string, decorator Decorator[T]) {
	key := reflect.TypeKeyNamed[T](name)

	c.internal.AddDecorator(
		key, func(ctx context.Context, r container.Resolver, instance any) (any, error) {
			typed, ok := instance.(T)
			if !ok {
				var zero T
				return zero, errDecoratorTypeMismatch(reflect.TypeName[T]())
			}

			resolver := &resolverAdapter{container: c}
			return decorator(ctx, resolver, typed)
		},
	)
}

func errDecoratorTypeMismatch(typeName string) *Error {
	return newError(
		ErrCodeDecoratorFailed,
		"decorator type mismatch for "+typeName,
		nil,
	)
}
