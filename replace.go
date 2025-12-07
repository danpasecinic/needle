package needle

import (
	"context"
	"fmt"
	reflectPkg "reflect"

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

func ReplaceFunc[T any](c *Container, constructor any, opts ...ProviderOption) error {
	params, returnType, err := reflect.FuncParams(constructor)
	if err != nil {
		return err
	}

	if returnType == nil {
		return fmt.Errorf("constructor must return at least one value")
	}

	fnVal := reflectPkg.ValueOf(constructor)
	fnType := fnVal.Type()

	hasError := fnType.NumOut() == 2 &&
		fnType.Out(1).Implements(reflectPkg.TypeOf((*error)(nil)).Elem())

	deps := make([]string, len(params))
	for i, p := range params {
		deps[i] = p.TypeKey
	}

	provider := func(ctx context.Context, r Resolver) (T, error) {
		var zero T

		args := make([]reflectPkg.Value, len(params))
		for i, p := range params {
			instance, err := c.internal.Resolve(ctx, p.TypeKey)
			if err != nil {
				return zero, fmt.Errorf("failed to resolve parameter %d (%s): %w", i, p.TypeKey, err)
			}
			args[i] = reflectPkg.ValueOf(instance)
		}

		results := fnVal.Call(args)

		if hasError && len(results) == 2 && !results[1].IsNil() {
			return zero, results[1].Interface().(error)
		}

		return results[0].Interface().(T), nil
	}

	opts = append([]ProviderOption{WithDependencies(deps...)}, opts...)
	return Replace(c, provider, opts...)
}

func MustReplaceFunc[T any](c *Container, constructor any, opts ...ProviderOption) {
	if err := ReplaceFunc[T](c, constructor, opts...); err != nil {
		panic(err)
	}
}

func ReplaceStruct[T any](c *Container, opts ...ProviderOption) error {
	provider := func(ctx context.Context, r Resolver) (T, error) {
		return InvokeStructCtx[T](ctx, c)
	}

	fields, _ := reflect.StructFields[T](TagKey)
	deps := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.Optional {
			if f.Named != "" {
				deps = append(deps, f.TypeKey+"#"+f.Named)
			} else {
				deps = append(deps, f.TypeKey)
			}
		}
	}

	opts = append([]ProviderOption{WithDependencies(deps...)}, opts...)
	return Replace(c, provider, opts...)
}

func MustReplaceStruct[T any](c *Container, opts ...ProviderOption) {
	if err := ReplaceStruct[T](c, opts...); err != nil {
		panic(err)
	}
}
