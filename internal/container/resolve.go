package container

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/danpasecinic/needle/internal/scope"
)

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

	for _, hook := range entry.OnStart {
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
