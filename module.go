package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

type Module struct {
	name       string
	registers  []func(c *Container) error
	decorators []decoratorEntry
	submodules []*Module
}

type decoratorEntry struct {
	key       string
	decorator func(ctx context.Context, r Resolver, instance any) (any, error)
}

func NewModule(name string) *Module {
	return &Module{
		name: name,
	}
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) Include(submodule *Module) *Module {
	m.submodules = append(m.submodules, submodule)
	return m
}

func ModuleRegister[T any](m *Module, spec Spec[T]) *Module {
	m.registers = append(m.registers, func(c *Container) error {
		return Register(c, spec)
	})
	return m
}

func ModuleReplace[T any](m *Module, spec Spec[T]) *Module {
	m.registers = append(m.registers, func(c *Container) error {
		return Replace(c, spec)
	})
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

func (m *Module) apply(c *Container) error {
	for _, sub := range m.submodules {
		if err := sub.apply(c); err != nil {
			return err
		}
	}

	for _, register := range m.registers {
		if err := register(c); err != nil {
			return err
		}
	}

	for _, d := range m.decorators {
		entry := d
		c.internal.AddDecorator(
			entry.key, func(ctx context.Context, _ container.Resolver, instance any) (any, error) {
				return entry.decorator(ctx, c.resolver, instance)
			},
		)
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
