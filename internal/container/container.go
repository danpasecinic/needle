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

	for _, hook := range c.onProvide {
		hook(key)
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

	for _, hook := range c.onProvide {
		hook(key)
	}

	return nil
}

func (c *Container) Resolve(ctx context.Context, key string) (any, error) {
	start := time.Now()

	c.resolvingMu.Lock()
	if c.resolving[key] {
		c.resolvingMu.Unlock()
		err := fmt.Errorf("circular resolution detected for: %s", key)
		c.callResolveHooks(key, time.Since(start), err)
		return nil, err
	}
	c.resolving[key] = true
	c.resolvingMu.Unlock()

	defer func() {
		c.resolvingMu.Lock()
		delete(c.resolving, key)
		c.resolvingMu.Unlock()
	}()

	c.mu.RLock()
	entry, exists := c.registry.Get(key)
	c.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("service not found: %s", key)
		c.callResolveHooks(key, time.Since(start), err)
		return nil, err
	}

	result, err := c.resolveWithScope(ctx, key, entry)
	c.callResolveHooks(key, time.Since(start), err)
	return result, err
}

func (c *Container) callResolveHooks(key string, duration time.Duration, err error) {
	for _, hook := range c.onResolve {
		hook(key, duration, err)
	}
}

func (c *Container) resolveWithScope(ctx context.Context, key string, entry *ServiceEntry) (any, error) {
	switch entry.Scope {
	case scope.Singleton:
		return c.resolveSingleton(ctx, key, entry)
	case scope.Transient:
		return c.resolveTransient(ctx, key, entry)
	case scope.Request:
		return c.resolveRequest(ctx, key, entry)
	case scope.Pooled:
		return c.resolvePooled(ctx, key, entry)
	default:
		return c.resolveSingleton(ctx, key, entry)
	}
}

func (c *Container) resolveSingleton(ctx context.Context, key string, entry *ServiceEntry) (any, error) {
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

	instance, err = c.applyDecorators(ctx, key, instance)
	if err != nil {
		return nil, err
	}

	c.registry.SetInstance(key, instance)
	return instance, nil
}

func (c *Container) resolveTransient(ctx context.Context, key string, entry *ServiceEntry) (any, error) {
	for _, dep := range entry.Dependencies {
		if _, err := c.Resolve(ctx, dep); err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s for %s: %w", dep, key, err)
		}
	}

	instance, err := entry.Provider(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("provider failed for %s: %w", key, err)
	}

	return c.applyDecorators(ctx, key, instance)
}

type requestScopeKey struct{}

type RequestScope struct {
	mu        sync.RWMutex
	instances map[string]any
}

func NewRequestScope() *RequestScope {
	return &RequestScope{
		instances: make(map[string]any),
	}
}

func (rs *RequestScope) Get(key string) (any, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	instance, ok := rs.instances[key]
	return instance, ok
}

func (rs *RequestScope) Set(key string, instance any) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.instances[key] = instance
}

func WithRequestScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestScopeKey{}, NewRequestScope())
}

func getRequestScope(ctx context.Context) *RequestScope {
	if rs, ok := ctx.Value(requestScopeKey{}).(*RequestScope); ok {
		return rs
	}
	return nil
}

func (c *Container) resolveRequest(ctx context.Context, key string, entry *ServiceEntry) (any, error) {
	rs := getRequestScope(ctx)
	if rs == nil {
		return nil, fmt.Errorf("request scope not found in context for %s; use WithRequestScope(ctx)", key)
	}

	if instance, ok := rs.Get(key); ok {
		return instance, nil
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

	instance, err = c.applyDecorators(ctx, key, instance)
	if err != nil {
		return nil, err
	}

	rs.Set(key, instance)
	return instance, nil
}

func (c *Container) resolvePooled(ctx context.Context, key string, entry *ServiceEntry) (any, error) {
	if instance, ok := c.registry.AcquireFromPool(key); ok {
		return instance, nil
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

	return c.applyDecorators(ctx, key, instance)
}

func (c *Container) Release(key string, instance any) bool {
	return c.registry.ReleaseToPool(key, instance)
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
		start := time.Now()

		if _, err := c.Resolve(ctx, key); err != nil {
			c.callStartHooks(key, time.Since(start), err)
			return fmt.Errorf("failed to resolve %s during startup: %w", key, err)
		}

		entry, exists := c.registry.GetEntry(key)
		if !exists {
			continue
		}

		var startErr error
		for _, hook := range entry.OnStart {
			c.logger.Debug("running OnStart hook", "service", key)
			if err := hook(ctx); err != nil {
				startErr = fmt.Errorf("OnStart hook failed for %s: %w", key, err)
				break
			}
		}

		c.callStartHooks(key, time.Since(start), startErr)
		if startErr != nil {
			return startErr
		}
	}

	c.mu.Lock()
	c.state = StateRunning
	c.mu.Unlock()

	return nil
}

func (c *Container) callStartHooks(key string, duration time.Duration, err error) {
	for _, hook := range c.onStart {
		hook(key, duration, err)
	}
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

		start := time.Now()
		var stopErr error

		for i := len(entry.OnStop) - 1; i >= 0; i-- {
			c.logger.Debug("running OnStop hook", "service", key)
			if err := entry.OnStop[i](ctx); err != nil {
				stopErr = fmt.Errorf("OnStop hook failed for %s: %w", key, err)
				errs = append(errs, stopErr)
			}
		}

		c.callStopHooks(key, time.Since(start), stopErr)
	}

	c.mu.Lock()
	c.state = StateStopped
	c.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

func (c *Container) callStopHooks(key string, duration time.Duration, err error) {
	for _, hook := range c.onStop {
		hook(key, duration, err)
	}
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

func (c *Container) AddDecorator(key string, decorator DecoratorFunc) {
	c.decoratorsMu.Lock()
	defer c.decoratorsMu.Unlock()

	c.decorators[key] = append(c.decorators[key], decorator)
}

func (c *Container) applyDecorators(ctx context.Context, key string, instance any) (any, error) {
	c.decoratorsMu.RLock()
	decorators := c.decorators[key]
	c.decoratorsMu.RUnlock()

	if len(decorators) == 0 {
		return instance, nil
	}

	var err error
	for _, decorator := range decorators {
		instance, err = decorator(ctx, c, instance)
		if err != nil {
			return nil, fmt.Errorf("decorator failed for %s: %w", key, err)
		}
	}

	return instance, nil
}
