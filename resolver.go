package needle

import (
	"context"
	"errors"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

func Invoke[T any](c *Container) (T, error) {
	return InvokeCtx[T](context.Background(), c)
}

func InvokeCtx[T any](ctx context.Context, c *Container) (T, error) {
	var zero T
	key := reflect.TypeKey[T]()

	instance, err := c.internal.Resolve(ctx, key)
	if err != nil {
		return zero, errResolutionFailed(reflect.TypeName[T](), err)
	}

	typed, ok := instance.(T)
	if !ok {
		return zero, errResolutionFailed(reflect.TypeName[T](), nil)
	}

	return typed, nil
}

func InvokeNamed[T any](c *Container, name string) (T, error) {
	return InvokeNamedCtx[T](context.Background(), c, name)
}

func InvokeNamedCtx[T any](ctx context.Context, c *Container, name string) (T, error) {
	var zero T
	key := reflect.TypeKeyNamed[T](name)

	instance, err := c.internal.Resolve(ctx, key)
	if err != nil {
		return zero, errResolutionFailed(reflect.TypeName[T]()+"#"+name, err)
	}

	typed, ok := instance.(T)
	if !ok {
		return zero, errResolutionFailed(reflect.TypeName[T]()+"#"+name, nil)
	}

	return typed, nil
}

func MustInvoke[T any](c *Container) T {
	v, err := Invoke[T](c)
	if err != nil {
		panic(err)
	}
	return v
}

func MustInvokeCtx[T any](ctx context.Context, c *Container) T {
	v, err := InvokeCtx[T](ctx, c)
	if err != nil {
		panic(err)
	}
	return v
}

func MustInvokeNamed[T any](c *Container, name string) T {
	v, err := InvokeNamed[T](c, name)
	if err != nil {
		panic(err)
	}
	return v
}

func MustInvokeNamedCtx[T any](ctx context.Context, c *Container, name string) T {
	v, err := InvokeNamedCtx[T](ctx, c, name)
	if err != nil {
		panic(err)
	}
	return v
}

func TryInvoke[T any](c *Container) (T, bool) {
	v, err := Invoke[T](c)
	return v, err == nil
}

func TryInvokeNamed[T any](c *Container, name string) (T, bool) {
	v, err := InvokeNamed[T](c, name)
	return v, err == nil
}

func Has[T any](c *Container) bool {
	key := reflect.TypeKey[T]()
	return c.internal.Has(key)
}

func HasNamed[T any](c *Container, name string) bool {
	key := reflect.TypeKeyNamed[T](name)
	return c.internal.Has(key)
}

type Optional[T any] struct {
	value   T
	present bool
}

func (o Optional[T]) Get() (T, bool) {
	return o.value, o.present
}

func (o Optional[T]) Value() T {
	return o.value
}

func (o Optional[T]) Present() bool {
	return o.present
}

func (o Optional[T]) OrElse(defaultValue T) T {
	if o.present {
		return o.value
	}
	return defaultValue
}

func (o Optional[T]) OrElseFunc(fn func() T) T {
	if o.present {
		return o.value
	}
	return fn()
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, present: true}
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func InvokeOptional[T any](c *Container) (Optional[T], error) {
	return InvokeOptionalCtx[T](context.Background(), c)
}

func InvokeOptionalCtx[T any](ctx context.Context, c *Container) (Optional[T], error) {
	return resolveOptional[T](ctx, c, reflect.TypeKey[T](), reflect.TypeName[T]())
}

func InvokeOptionalNamed[T any](c *Container, name string) (Optional[T], error) {
	return InvokeOptionalNamedCtx[T](context.Background(), c, name)
}

func InvokeOptionalNamedCtx[T any](ctx context.Context, c *Container, name string) (Optional[T], error) {
	return resolveOptional[T](ctx, c, reflect.TypeKeyNamed[T](name), reflect.TypeName[T]()+"#"+name)
}

func resolveOptional[T any](ctx context.Context, c *Container, key, displayName string) (Optional[T], error) {
	instance, err := c.internal.Resolve(ctx, key)
	if err != nil {
		if errors.Is(err, container.ErrServiceNotFound) {
			return None[T](), nil
		}
		return None[T](), errResolutionFailed(displayName, err)
	}

	typed, ok := instance.(T)
	if !ok {
		return None[T](), errResolutionFailed(displayName, nil)
	}

	return Some(typed), nil
}
