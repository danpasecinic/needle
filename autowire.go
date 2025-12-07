package needle

import (
	"context"
	"fmt"
	reflectPkg "reflect"

	"github.com/danpasecinic/needle/internal/reflect"
)

const TagKey = "needle"

func InvokeStruct[T any](c *Container) (T, error) {
	return InvokeStructCtx[T](context.Background(), c)
}

func InvokeStructCtx[T any](ctx context.Context, c *Container) (T, error) {
	var zero T

	t := reflectPkg.TypeOf(zero)
	isPtr := t.Kind() == reflectPkg.Ptr
	if isPtr {
		t = t.Elem()
	}

	if t.Kind() != reflectPkg.Struct {
		return zero, fmt.Errorf("InvokeStruct requires a struct type, got %s", t.Kind())
	}

	fields, err := reflect.StructFields[T](TagKey)
	if err != nil {
		return zero, err
	}

	structVal := reflectPkg.New(t).Elem()

	for _, field := range fields {
		var key string
		if field.Named != "" {
			key = field.TypeKey + "#" + field.Named
		} else {
			key = field.TypeKey
		}

		if !c.internal.Has(key) {
			if field.Optional {
				continue
			}
			return zero, errServiceNotFound(key)
		}

		instance, err := c.internal.Resolve(ctx, key)
		if err != nil {
			if field.Optional {
				continue
			}
			return zero, errResolutionFailed(field.Name, err)
		}

		fieldVal := structVal.Field(field.Index)
		if !fieldVal.CanSet() {
			return zero, fmt.Errorf("cannot set field %s (unexported)", field.Name)
		}

		instanceVal := reflectPkg.ValueOf(instance)
		if !instanceVal.Type().AssignableTo(fieldVal.Type()) {
			return zero, fmt.Errorf(
				"cannot assign %s to field %s of type %s",
				instanceVal.Type(), field.Name, fieldVal.Type(),
			)
		}

		fieldVal.Set(instanceVal)
	}

	if isPtr {
		ptr := reflectPkg.New(t)
		ptr.Elem().Set(structVal)
		return ptr.Interface().(T), nil
	}

	return structVal.Interface().(T), nil
}

func ProvideFunc[T any](c *Container, constructor any, opts ...ProviderOption) error {
	params, returnType, err := reflect.FuncParams(constructor)
	if err != nil {
		return err
	}

	if returnType == nil {
		return fmt.Errorf("constructor must return at least one value")
	}

	expectedType := reflectPkg.TypeOf((*T)(nil)).Elem()
	if !returnType.AssignableTo(expectedType) {
		return fmt.Errorf("constructor returns %s, expected %s", returnType, expectedType)
	}

	fnVal := reflectPkg.ValueOf(constructor)
	fnType := fnVal.Type()

	hasError := fnType.NumOut() == 2 && fnType.Out(1).Implements(reflectPkg.TypeOf((*error)(nil)).Elem())

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
	return Provide(c, provider, opts...)
}

func MustProvideFunc[T any](c *Container, constructor any, opts ...ProviderOption) {
	if err := ProvideFunc[T](c, constructor, opts...); err != nil {
		panic(err)
	}
}

func ProvideStruct[T any](c *Container, opts ...ProviderOption) error {
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
	return Provide(c, provider, opts...)
}

func MustProvideStruct[T any](c *Container, opts ...ProviderOption) {
	if err := ProvideStruct[T](c, opts...); err != nil {
		panic(err)
	}
}
