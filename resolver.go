package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/reflect"
)

type Resolver interface {
	Resolve(ctx context.Context, key string) (any, error)
	Has(key string) bool
}

type resolverAdapter struct {
	container *Container
}

func (r *resolverAdapter) Resolve(ctx context.Context, key string) (any, error) {
	return r.container.internal.Resolve(ctx, key)
}

func (r *resolverAdapter) Has(key string) bool {
	return r.container.internal.Has(key)
}

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
