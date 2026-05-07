package needle

import (
	"context"
	"errors"
	"fmt"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type Provider[T any] func(ctx context.Context, r Resolver) (T, error)

type Spec[T any] struct {
	Name         string
	Provider     Provider[T]
	Dependencies []string
	Scope        Scope
	OnStart      Hook
	OnStop       Hook
	PoolSize     int
	Lazy         bool

	value    T
	hasValue bool
}

func SpecValue[T any](v T) Spec[T] {
	return Spec[T]{value: v, hasValue: true}
}

func (s Spec[T]) WithName(name string) Spec[T] {
	s.Name = name
	return s
}

func (s Spec[T]) WithDependencies(deps ...string) Spec[T] {
	s.Dependencies = deps
	return s
}

func (s Spec[T]) WithScope(sc Scope) Spec[T] {
	s.Scope = sc
	return s
}

func (s Spec[T]) WithPoolSize(n int) Spec[T] {
	s.Scope = Pooled
	s.PoolSize = n
	return s
}

func (s Spec[T]) WithLazy() Spec[T] {
	s.Lazy = true
	return s
}

func (s Spec[T]) WithOnStart(hook Hook) Spec[T] {
	s.OnStart = hook
	return s
}

func (s Spec[T]) WithOnStop(hook Hook) Spec[T] {
	s.OnStop = hook
	return s
}

func Compose(hooks ...Hook) Hook {
	switch len(hooks) {
	case 0:
		return nil
	case 1:
		return hooks[0]
	}
	return func(ctx context.Context) error {
		for _, h := range hooks {
			if h == nil {
				continue
			}
			if err := h(ctx); err != nil {
				return err
			}
		}
		return nil
	}
}

func Register[T any](c *Container, spec Spec[T]) error {
	entry, err := buildEntry(c, spec)
	if err != nil {
		return err
	}
	return c.internal.Register(entry)
}

func MustRegister[T any](c *Container, spec Spec[T]) {
	if err := Register(c, spec); err != nil {
		panic(err)
	}
}

func Replace[T any](c *Container, spec Spec[T]) error {
	entry, err := buildEntry(c, spec)
	if err != nil {
		return err
	}
	return c.internal.Replace(entry)
}

func MustReplace[T any](c *Container, spec Spec[T]) {
	if err := Replace(c, spec); err != nil {
		panic(err)
	}
}

var (
	errSpecBothProviderAndValue       = errors.New("spec has both Provider and Value set; exactly one must be provided")
	errSpecNeitherProviderNorValue    = errors.New("spec has neither Provider nor Value set; exactly one must be provided")
	errSpecPoolSizeWithNonPooledScope = errors.New("spec has PoolSize > 0 but Scope is not Pooled; use WithPoolSize or set Scope = Pooled")
	errSpecValueWithLazy              = errors.New("spec has a Value and Lazy = true; values are eagerly bound and cannot be lazy")
	errSpecValueWithNonSingletonScope = errors.New("spec has a Value with non-singleton Scope; values are inherently singleton")
)

func buildEntry[T any](c *Container, spec Spec[T]) (*container.ServiceEntry, error) {
	if spec.Provider != nil && spec.hasValue {
		return nil, errSpecBothProviderAndValue
	}
	if spec.Provider == nil && !spec.hasValue {
		return nil, errSpecNeitherProviderNorValue
	}
	if spec.hasValue {
		if spec.Lazy {
			return nil, errSpecValueWithLazy
		}
		if spec.Scope != Singleton {
			return nil, errSpecValueWithNonSingletonScope
		}
		if spec.PoolSize > 0 {
			return nil, errSpecPoolSizeWithNonPooledScope
		}
	}
	if spec.PoolSize > 0 && spec.Scope != Pooled {
		return nil, errSpecPoolSizeWithNonPooledScope
	}

	key := reflect.TypeKey[T]()
	if spec.Name != "" {
		key = reflect.TypeKeyNamed[T](spec.Name)
	}

	cfg := container.EntryConfig{
		Key:          key,
		Dependencies: spec.Dependencies,
		Scope:        spec.Scope,
		PoolSize:     spec.PoolSize,
		Lazy:         spec.Lazy,
		OnStart:      container.Hook(spec.OnStart),
		OnStop:       container.Hook(spec.OnStop),
	}

	if spec.hasValue {
		cfg.Value = spec.value
		cfg.HasValue = true
	} else {
		provider := spec.Provider
		resolver := c.resolver
		cfg.Provider = func(ctx context.Context, _ container.Resolver) (any, error) {
			return provider(ctx, resolver)
		}
	}

	return container.NewServiceEntry(cfg), nil
}

// SpecFromConstructor builds a spec whose Provider auto-resolves the constructor's
// parameters from the resolver and calls the constructor. The constructor must
// return T (and optionally an error as the second return value). Panics if the
// constructor signature is invalid – a programmer error caught at startup.
func SpecFromConstructor[T any](constructor any) Spec[T] {
	provider, deps, err := buildFuncProvider[T](constructor)
	if err != nil {
		panic(fmt.Errorf("needle: SpecFromConstructor: %w", err))
	}
	return Spec[T]{
		Provider:     provider,
		Dependencies: deps,
	}
}

// SpecFromStruct builds a spec whose Provider populates a T (or *T) struct from
// fields tagged `needle:"..."`. Optional fields are silently skipped if absent.
func SpecFromStruct[T any]() Spec[T] {
	provider, deps := buildStructProvider[T]()
	return Spec[T]{
		Provider:     provider,
		Dependencies: deps,
	}
}

// SpecFromBinding builds a spec for interface I that resolves implementation T
// from the container and returns it as I. Use to wire an interface to a concrete
// type already registered under T's key.
func SpecFromBinding[I, T any]() Spec[I] {
	implKey := reflect.TypeKey[T]()
	return Spec[I]{
		Provider: func(ctx context.Context, r Resolver) (I, error) {
			var zero I
			instance, err := r.Resolve(ctx, implKey)
			if err != nil {
				return zero, err
			}
			typed, ok := instance.(I)
			if !ok {
				return zero, fmt.Errorf("binding type mismatch: %s does not implement %s", reflect.TypeName[T](), reflect.TypeName[I]())
			}
			return typed, nil
		},
		Dependencies: []string{implKey},
	}
}
