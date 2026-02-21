package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/danpasecinic/needle/internal/scope"
)

type resolvingKey struct{}

func withResolving(ctx context.Context, key string) (context.Context, bool) {
	set, _ := ctx.Value(resolvingKey{}).(map[string]bool)
	if set != nil && set[key] {
		return ctx, false
	}
	if set == nil {
		set = make(map[string]bool)
	}
	next := make(map[string]bool, len(set)+1)
	for k := range set {
		next[k] = true
	}
	next[key] = true
	return context.WithValue(ctx, resolvingKey{}, next), true
}

func (c *Container) Resolve(ctx context.Context, key string) (any, error) {
	if len(c.onResolve) == 0 {
		if instance, ok := c.registry.GetInstanceFast(key); ok {
			return instance, nil
		}
	}

	return c.resolveSlow(ctx, key)
}

func (c *Container) resolveSlow(ctx context.Context, key string) (any, error) {
	start := time.Now()

	ctx, ok := withResolving(ctx, key)
	if !ok {
		err := fmt.Errorf("circular resolution detected for: %s", key)
		c.callResolveHooks(key, time.Since(start), err)
		return nil, err
	}

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

	var instance any
	entry.once.Do(func() {
		for _, dep := range entry.Dependencies {
			if _, err := c.Resolve(ctx, dep); err != nil {
				entry.initErr = fmt.Errorf("failed to resolve dependency %s for %s: %w", dep, key, err)
				return
			}
		}

		inst, err := entry.Provider(ctx, c)
		if err != nil {
			entry.initErr = fmt.Errorf("provider failed for %s: %w", key, err)
			return
		}

		inst, err = c.applyDecorators(ctx, key, inst)
		if err != nil {
			entry.initErr = err
			return
		}

		c.registry.SetInstance(key, inst)
	})

	if entry.initErr != nil {
		return nil, entry.initErr
	}

	instance = entry.Instance

	if entry.Lazy && !entry.StartRan && c.state == StateRunning {
		if err := c.runLazyStart(ctx, key, entry); err != nil {
			return nil, err
		}
	}

	return instance, nil
}

func (c *Container) runLazyStart(ctx context.Context, key string, entry *ServiceEntry) error {
	start := time.Now()
	var startErr error

	hooks := c.registry.GetOnStartHooks(key)
	for _, hook := range hooks {
		c.logger.Debug("running lazy OnStart hook", "service", key)
		if err := hook(ctx); err != nil {
			startErr = fmt.Errorf("OnStart hook failed for %s: %w", key, err)
			break
		}
	}

	c.registry.SetStartRan(key)
	c.callStartHooks(key, time.Since(start), startErr)
	return startErr
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
