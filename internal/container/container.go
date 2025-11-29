package container

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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

type Container struct {
	mu       sync.RWMutex
	registry *Registry
	graph    *graph.Graph
	logger   *slog.Logger
	state    State

	resolving   map[string]bool
	resolvingMu sync.Mutex
}

type Config struct {
	Logger *slog.Logger
}

func New(cfg *Config) *Container {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Container{
		registry:  NewRegistry(),
		graph:     graph.New(),
		logger:    logger,
		resolving: make(map[string]bool),
	}
}

func (c *Container) Register(key string, provider ProviderFunc, dependencies []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.registry.Has(key) {
		return fmt.Errorf("service already registered: %s", key)
	}

	if err := c.registry.Register(key, provider, dependencies); err != nil {
		return err
	}

	c.graph.AddNode(key, dependencies)

	if c.graph.HasCycle() {
		c.registry.Remove(key)
		c.graph.RemoveNode(key)
		cyclePath := c.graph.FindCyclePath(key)
		return fmt.Errorf("circular dependency detected: %v", cyclePath)
	}

	return nil
}

func (c *Container) RegisterValue(key string, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.registry.Has(key) {
		return fmt.Errorf("service already registered: %s", key)
	}

	if err := c.registry.RegisterValue(key, value); err != nil {
		return err
	}

	c.graph.AddNode(key, nil)
	return nil
}

func (c *Container) Resolve(ctx context.Context, key string) (any, error) {
	c.resolvingMu.Lock()
	if c.resolving[key] {
		c.resolvingMu.Unlock()
		return nil, fmt.Errorf("circular resolution detected for: %s", key)
	}
	c.resolving[key] = true
	c.resolvingMu.Unlock()

	defer func() {
		c.resolvingMu.Lock()
		delete(c.resolving, key)
		c.resolvingMu.Unlock()
	}()

	if instance, ok := c.registry.GetInstance(key); ok {
		return instance, nil
	}

	c.mu.RLock()
	entry, exists := c.registry.Get(key)
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service not found: %s", key)
	}

	if entry.Instantiated {
		return entry.Instance, nil
	}

	for _, dep := range entry.Dependencies {
		if _, err := c.Resolve(ctx, dep); err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s for %s: %w", dep, key, err)
		}
	}

	instance, err := entry.Provider(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("provider failed for %s: %w", key, err)
	}

	c.registry.SetInstance(key, instance)
	return instance, nil
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

func (c *Container) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateNew && c.state != StateStopped {
		c.mu.Unlock()
		return fmt.Errorf("container already started")
	}
	c.state = StateStarting
	c.mu.Unlock()

	order, err := c.graph.StartupOrder()
	if err != nil {
		return fmt.Errorf("failed to determine startup order: %w", err)
	}

	for _, key := range order {
		if _, err := c.Resolve(ctx, key); err != nil {
			return fmt.Errorf("failed to resolve %s during startup: %w", key, err)
		}

		entry, exists := c.registry.GetEntry(key)
		if !exists {
			continue
		}

		for _, hook := range entry.OnStart {
			c.logger.Debug("running OnStart hook", "service", key)
			if err := hook(ctx); err != nil {
				return fmt.Errorf("OnStart hook failed for %s: %w", key, err)
			}
		}
	}

	c.mu.Lock()
	c.state = StateRunning
	c.mu.Unlock()

	return nil
}

func (c *Container) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateRunning {
		c.mu.Unlock()
		return nil
	}
	c.state = StateStopping
	c.mu.Unlock()

	order, err := c.graph.ShutdownOrder()
	if err != nil {
		return fmt.Errorf("failed to determine shutdown order: %w", err)
	}

	var errs []error
	for _, key := range order {
		entry, exists := c.registry.GetEntry(key)
		if !exists || !entry.Instantiated {
			continue
		}

		for i := len(entry.OnStop) - 1; i >= 0; i-- {
			c.logger.Debug("running OnStop hook", "service", key)
			if err := entry.OnStop[i](ctx); err != nil {
				errs = append(errs, fmt.Errorf("OnStop hook failed for %s: %w", key, err))
			}
		}
	}

	c.mu.Lock()
	c.state = StateStopped
	c.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

func (c *Container) AddOnStart(key string, hook Hook) {
	c.registry.AddOnStart(key, hook)
}

func (c *Container) AddOnStop(key string, hook Hook) {
	c.registry.AddOnStop(key, hook)
}
