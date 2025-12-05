package needletest

import (
	"context"

	"github.com/danpasecinic/needle"
	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type TB interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Cleanup(f func())
}

type TestContainer struct {
	*needle.Container
	tb TB
}

func New(tb TB, opts ...needle.Option) *TestContainer {
	tb.Helper()

	c := needle.New(opts...)
	tc := &TestContainer{
		Container: c,
		tb:        tb,
	}

	tb.Cleanup(func() {
		if err := c.Stop(context.Background()); err != nil {
			tb.Fatalf("failed to stop container: %v", err)
		}
	})

	return tc
}

func (tc *TestContainer) RequireStart(ctx context.Context) {
	tc.tb.Helper()

	if err := tc.Start(ctx); err != nil {
		tc.tb.Fatalf("failed to start container: %v", err)
	}
}

func (tc *TestContainer) RequireStop(ctx context.Context) {
	tc.tb.Helper()

	if err := tc.Stop(ctx); err != nil {
		tc.tb.Fatalf("failed to stop container: %v", err)
	}
}

func (tc *TestContainer) RequireValidate() {
	tc.tb.Helper()

	if err := tc.Validate(); err != nil {
		tc.tb.Fatalf("container validation failed: %v", err)
	}
}

func Replace[T any](tc *TestContainer, value T) {
	tc.tb.Helper()

	key := reflect.TypeKey[T]()
	if err := tc.Container.Internal().ReplaceValue(key, value); err != nil {
		tc.tb.Fatalf("failed to replace %s: %v", key, err)
	}
}

func ReplaceNamed[T any](tc *TestContainer, name string, value T) {
	tc.tb.Helper()

	key := reflect.TypeKeyNamed[T](name)
	if err := tc.Container.Internal().ReplaceValue(key, value); err != nil {
		tc.tb.Fatalf("failed to replace %s: %v", key, err)
	}
}

func ReplaceProvider[T any](tc *TestContainer, provider needle.Provider[T]) {
	tc.tb.Helper()

	key := reflect.TypeKey[T]()
	wrappedProvider := func(ctx context.Context, r container.Resolver) (any, error) {
		return provider(ctx, nil)
	}

	if err := tc.Container.Internal().Replace(key, wrappedProvider, nil); err != nil {
		tc.tb.Fatalf("failed to replace provider %s: %v", key, err)
	}
}

func ReplaceNamedProvider[T any](tc *TestContainer, name string, provider needle.Provider[T]) {
	tc.tb.Helper()

	key := reflect.TypeKeyNamed[T](name)
	wrappedProvider := func(ctx context.Context, r container.Resolver) (any, error) {
		return provider(ctx, nil)
	}

	if err := tc.Container.Internal().Replace(key, wrappedProvider, nil); err != nil {
		tc.tb.Fatalf("failed to replace provider %s: %v", key, err)
	}
}

func AssertHas[T any](tc *TestContainer) {
	tc.tb.Helper()

	if !needle.Has[T](tc.Container) {
		tc.tb.Fatalf("expected container to have %s", reflect.TypeKey[T]())
	}
}

func AssertHasNamed[T any](tc *TestContainer, name string) {
	tc.tb.Helper()

	if !needle.HasNamed[T](tc.Container, name) {
		tc.tb.Fatalf("expected container to have %s", reflect.TypeKeyNamed[T](name))
	}
}

func AssertNotHas[T any](tc *TestContainer) {
	tc.tb.Helper()

	if needle.Has[T](tc.Container) {
		tc.tb.Fatalf("expected container to not have %s", reflect.TypeKey[T]())
	}
}

func MustInvoke[T any](tc *TestContainer) T {
	tc.tb.Helper()

	v, err := needle.Invoke[T](tc.Container)
	if err != nil {
		tc.tb.Fatalf("failed to invoke %s: %v", reflect.TypeKey[T](), err)
	}
	return v
}

func MustInvokeNamed[T any](tc *TestContainer, name string) T {
	tc.tb.Helper()

	v, err := needle.InvokeNamed[T](tc.Container, name)
	if err != nil {
		tc.tb.Fatalf("failed to invoke %s: %v", reflect.TypeKeyNamed[T](name), err)
	}
	return v
}

func MustProvide[T any](tc *TestContainer, provider needle.Provider[T], opts ...needle.ProviderOption) {
	tc.tb.Helper()

	if err := needle.Provide(tc.Container, provider, opts...); err != nil {
		tc.tb.Fatalf("failed to provide %s: %v", reflect.TypeKey[T](), err)
	}
}

func MustProvideValue[T any](tc *TestContainer, value T, opts ...needle.ProviderOption) {
	tc.tb.Helper()

	if err := needle.ProvideValue(tc.Container, value, opts...); err != nil {
		tc.tb.Fatalf("failed to provide value %s: %v", reflect.TypeKey[T](), err)
	}
}

func MustProvideNamed[T any](tc *TestContainer, name string, provider needle.Provider[T], opts ...needle.ProviderOption) {
	tc.tb.Helper()

	if err := needle.ProvideNamed(tc.Container, name, provider, opts...); err != nil {
		tc.tb.Fatalf("failed to provide %s: %v", reflect.TypeKeyNamed[T](name), err)
	}
}

func MustProvideNamedValue[T any](tc *TestContainer, name string, value T, opts ...needle.ProviderOption) {
	tc.tb.Helper()

	if err := needle.ProvideNamedValue(tc.Container, name, value, opts...); err != nil {
		tc.tb.Fatalf("failed to provide value %s: %v", reflect.TypeKeyNamed[T](name), err)
	}
}
