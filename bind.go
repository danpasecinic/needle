package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type Decorator[T any] func(ctx context.Context, r Resolver, base T) (T, error)

func Decorate[T any](c *Container, decorator Decorator[T]) {
	decorateKey(c, reflect.TypeKey[T](), decorator)
}

func DecorateNamed[T any](c *Container, name string, decorator Decorator[T]) {
	decorateKey(c, reflect.TypeKeyNamed[T](name), decorator)
}

func decorateKey[T any](c *Container, key string, decorator Decorator[T]) {
	c.internal.AddDecorator(
		key, func(ctx context.Context, _ container.Resolver, instance any) (any, error) {
			typed, ok := instance.(T)
			if !ok {
				var zero T
				return zero, errDecoratorTypeMismatch(reflect.TypeName[T]())
			}
			return decorator(ctx, c.resolver, typed)
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
