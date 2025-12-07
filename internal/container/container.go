package container

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/danpasecinic/needle/internal/graph"
	"github.com/danpasecinic/needle/internal/scope"
)

type State int

const (
	StateNew State = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
)

type DecoratorFunc func(ctx context.Context, r Resolver, instance any) (any, error)

type Container struct {
	mu       sync.RWMutex
	registry *Registry
	graph    *graph.Graph
	logger   *slog.Logger
	state    State

	resolving   map[string]bool
	resolvingMu sync.Mutex

	decorators   map[string][]DecoratorFunc
	decoratorsMu sync.RWMutex

	onResolve []ResolveHook
	onProvide []ProvideHook
	onStart   []StartHook
	onStop    []StopHook

	parallel bool
}

type ResolveHook func(key string, duration time.Duration, err error)
type ProvideHook func(key string)
type StartHook func(key string, duration time.Duration, err error)
type StopHook func(key string, duration time.Duration, err error)

type Config struct {
	Logger    *slog.Logger
	OnResolve []ResolveHook
	OnProvide []ProvideHook
	OnStart   []StartHook
	OnStop    []StopHook
	Parallel  bool
}

func New(cfg *Config) *Container {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Container{
		registry:   NewRegistry(),
		graph:      graph.New(),
		logger:     logger,
		resolving:  make(map[string]bool),
		decorators: make(map[string][]DecoratorFunc),
		onResolve:  cfg.OnResolve,
		onProvide:  cfg.OnProvide,
		onStart:    cfg.OnStart,
		onStop:     cfg.OnStop,
		parallel:   cfg.Parallel,
	}
}

func (c *Container) Register(key string, provider ProviderFunc, dependencies []string) error {
	c.mu.Lock()

	if c.registry.HasUnsafe(key) {
		c.mu.Unlock()
		return fmt.Errorf("service already registered: %s", key)
	}

	c.registry.RegisterUnsafe(key, provider, dependencies)
	c.graph.AddNodeUnsafe(key, dependencies)

	if c.graph.HasCycle() {
		c.registry.RemoveUnsafe(key)
		c.graph.RemoveNodeUnsafe(key)
		c.mu.Unlock()
		return fmt.Errorf("circular dependency detected for: %s", key)
	}

	c.mu.Unlock()

	for _, hook := range c.onProvide {
		hook(key)
	}

	return nil
}

func (c *Container) RegisterValue(key string, value any) error {
	c.mu.Lock()

	if c.registry.HasUnsafe(key) {
		c.mu.Unlock()
		return fmt.Errorf("service already registered: %s", key)
	}

	c.registry.RegisterValueUnsafe(key, value)
	c.graph.AddNodeUnsafe(key, nil)

	c.mu.Unlock()

	for _, hook := range c.onProvide {
		hook(key)
	}

	return nil
}

func (c *Container) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.registry.Has(key)
}

func (c *Container) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.registry.Keys()
}

func (c *Container) GetInstance(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.registry.GetInstance(key)
}

func (c *Container) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.registry.Size()
}

func (c *Container) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	missing := c.graph.Validate()
	if len(missing) > 0 {
		return fmt.Errorf("missing dependencies: %v", missing)
	}

	if c.graph.HasCycle() {
		cycles := c.graph.GetAllCyclePaths()
		return fmt.Errorf("circular dependencies detected: %v", cycles)
	}

	return nil
}

func (c *Container) Graph() *graph.Graph {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.graph.Clone()
}

func (c *Container) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Container) Release(key string, instance any) bool {
	return c.registry.ReleaseToPool(key, instance)
}

func (c *Container) AddOnStart(key string, hook Hook) {
	c.registry.AddOnStart(key, hook)
}

func (c *Container) AddOnStop(key string, hook Hook) {
	c.registry.AddOnStop(key, hook)
}

func (c *Container) SetScope(key string, s scope.Scope) {
	c.registry.SetScope(key, s)
}

func (c *Container) SetPoolSize(key string, size int) {
	c.registry.SetPoolSize(key, size)
}

func (c *Container) SetLazy(key string, lazy bool) {
	c.registry.SetLazy(key, lazy)
}
