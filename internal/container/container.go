package container

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/danpasecinic/needle/internal/graph"
)

type State int

const (
	StateNew State = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
)

type DecoratorFunc func(ctx context.Context, instance any) (any, error)

type Container struct {
	mu       sync.RWMutex
	registry *Registry
	graph    *graph.Graph
	logger   *slog.Logger
	state    State

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
		decorators: make(map[string][]DecoratorFunc),
		onResolve:  cfg.OnResolve,
		onProvide:  cfg.OnProvide,
		onStart:    cfg.OnStart,
		onStop:     cfg.OnStop,
		parallel:   cfg.Parallel,
	}
}

func (c *Container) Register(entry *ServiceEntry) error {
	if err := c.registerLocked(entry); err != nil {
		return err
	}

	for _, hook := range c.onProvide {
		hook(entry.Key)
	}

	return nil
}

func (c *Container) registerLocked(entry *ServiceEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.registry.Has(entry.Key) {
		return fmt.Errorf("%w: %s", ErrDuplicateService, entry.Key)
	}

	c.registry.Add(entry)
	c.graph.AddNode(entry.Key, entry.Dependencies)

	if len(entry.Dependencies) > 0 && c.graph.HasCycle() {
		c.registry.Remove(entry.Key)
		c.graph.RemoveNode(entry.Key)
		return fmt.Errorf("%w: %s", ErrCircularDependency, entry.Key)
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
		return fmt.Errorf("%w: %v", ErrCircularDependency, cycles)
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
	entry, exists := c.registry.GetEntry(key)
	if !exists {
		c.logger.Warn("pool overflow: instance dropped", "service", key)
		return false
	}
	if returnToPool(entry, instance) {
		return true
	}
	c.logger.Warn("pool overflow: instance dropped", "service", key)
	return false
}
