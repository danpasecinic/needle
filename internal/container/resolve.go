package container

import (
	"context"
	"fmt"
	"time"
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
		err := fmt.Errorf("%w: %s", ErrCircularResolution, key)
		c.callResolveHooks(key, time.Since(start), err)
		return nil, err
	}

	c.mu.RLock()
	entry, exists := c.registry.Get(key)
	c.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("%w: %s", ErrServiceNotFound, key)
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
	return strategyFor(entry.Scope).Acquire(ctx, c, key, entry, func() (any, error) {
		return c.buildInstance(ctx, key, entry)
	})
}

// buildInstance is the shared dep-resolve / provider-call / decorator chain
// used by every scope strategy. The strategy decides whether (and when) to
// call this; the loop itself lives in one place.
func (c *Container) buildInstance(ctx context.Context, key string, entry *ServiceEntry) (any, error) {
	for _, dep := range entry.Dependencies {
		if _, err := c.Resolve(ctx, dep); err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s for %s: %w", dep, key, err)
		}
	}

	instance, err := entry.Provider(ctx)
	if err != nil {
		return nil, fmt.Errorf("provider failed for %s: %w", key, err)
	}

	return c.applyDecorators(ctx, key, instance)
}

func (c *Container) runLazyStart(ctx context.Context, key string, entry *ServiceEntry) error {
	start := time.Now()
	var startErr error

	if entry.OnStart != nil {
		c.logger.Debug("running lazy OnStart hook", "service", key)
		if err := entry.OnStart(ctx); err != nil {
			startErr = fmt.Errorf("OnStart hook failed for %s: %w", key, err)
		}
	}

	c.registry.SetStartRan(key)
	c.callStartHooks(key, time.Since(start), startErr)
	return startErr
}
