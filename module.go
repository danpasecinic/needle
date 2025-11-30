package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type Module struct {
	name       string
	providers  []providerEntry
	decorators []decoratorEntry
	bindings   []bindingEntry
	submodules []*Module
}

type providerEntry struct {
	register func(c *Container) error
}

type decoratorEntry struct {
	key       string
	decorator func(ctx context.Context, r Resolver, instance any) (any, error)
}

type bindingEntry struct {
	interfaceKey string
	implKey      string
	opts         []ProviderOption
}

func NewModule(name string) *Module {
	return &Module{
		name: name,
	}
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) Provide(provider any, opts ...ProviderOption) *Module {
	m.providers = append(
		m.providers, providerEntry{
			register: func(c *Container) error {
				return provideAny(c, provider, opts...)
			},
		},
	)
	return m
}

func (m *Module) ProvideValue(value any, opts ...ProviderOption) *Module {
	m.providers = append(
		m.providers, providerEntry{
			register: func(c *Container) error {
				return provideValueAny(c, value, opts...)
			},
		},
	)
	return m
}

func (m *Module) Include(submodule *Module) *Module {
	m.submodules = append(m.submodules, submodule)
	return m
}

func (m *Module) apply(c *Container) error {
	for _, sub := range m.submodules {
		if err := sub.apply(c); err != nil {
			return err
		}
	}

	for _, p := range m.providers {
		if err := p.register(c); err != nil {
			return err
		}
	}

	for _, b := range m.bindings {
		if err := applyBinding(c, b); err != nil {
			return err
		}
	}

	for _, d := range m.decorators {
		c.internal.AddDecorator(
			d.key, func(ctx context.Context, r container.Resolver, instance any) (any, error) {
				resolver := &resolverAdapter{container: c}
				return d.decorator(ctx, resolver, instance)
			},
		)
	}

	return nil
}

func applyBinding(c *Container, b bindingEntry) error {
	cfg := &providerConfig{}
	for _, opt := range b.opts {
		opt(cfg)
	}

	key := b.interfaceKey
	if cfg.name != "" {
		key = cfg.name + "#" + b.interfaceKey
	}

	wrappedProvider := func(ctx context.Context, r container.Resolver) (any, error) {
		return r.Resolve(ctx, b.implKey)
	}

	if err := c.internal.Register(key, wrappedProvider, []string{b.implKey}); err != nil {
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

func provideAny(c *Container, provider any, opts ...ProviderOption) error {
	switch p := provider.(type) {
	case func(context.Context, Resolver) (any, error):
		return Provide(c, p, opts...)
	default:
		return errModuleInvalidProvider(provider)
	}
}

func provideValueAny(c *Container, value any, opts ...ProviderOption) error {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	key := reflect.TypeKeyFromValue(value)
	if cfg.name != "" {
		key = reflect.TypeKeyNamedFromValue(value, cfg.name)
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

func (c *Container) Apply(modules ...*Module) error {
	for _, m := range modules {
		if err := m.apply(c); err != nil {
			return errModuleApplyFailed(m.name, err)
		}
	}
	return nil
}

func errModuleApplyFailed(moduleName string, cause error) *Error {
	return newError(
		ErrCodeModuleApplyFailed,
		"failed to apply module "+moduleName,
		cause,
	)
}

func errModuleInvalidProvider(provider any) *Error {
	return newError(
		ErrCodeModuleInvalidProvider,
		"invalid provider type in module",
		nil,
	)
}

func ModuleProvide[T any](m *Module, provider Provider[T], opts ...ProviderOption) *Module {
	m.providers = append(
		m.providers, providerEntry{
			register: func(c *Container) error {
				return Provide(c, provider, opts...)
			},
		},
	)
	return m
}

func ModuleProvideValue[T any](m *Module, value T, opts ...ProviderOption) *Module {
	m.providers = append(
		m.providers, providerEntry{
			register: func(c *Container) error {
				return ProvideValue(c, value, opts...)
			},
		},
	)
	return m
}

func ModuleBind[I, T any](m *Module, opts ...ProviderOption) *Module {
	interfaceKey := reflect.TypeKey[I]()
	implKey := reflect.TypeKey[T]()

	m.bindings = append(
		m.bindings, bindingEntry{
			interfaceKey: interfaceKey,
			implKey:      implKey,
			opts:         opts,
		},
	)
	return m
}

func ModuleDecorate[T any](m *Module, decorator Decorator[T]) *Module {
	key := reflect.TypeKey[T]()

	m.decorators = append(
		m.decorators, decoratorEntry{
			key: key,
			decorator: func(ctx context.Context, r Resolver, instance any) (any, error) {
				typed, ok := instance.(T)
				if !ok {
					var zero T
					return zero, errDecoratorTypeMismatch(reflect.TypeName[T]())
				}
				return decorator(ctx, r, typed)
			},
		},
	)
	return m
}
