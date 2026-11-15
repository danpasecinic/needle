package container

import (
	"context"
	"fmt"
	"sync"

	"github.com/danpasecinic/needle/internal/scope"
)

// ScopeStrategy decides how to acquire an instance for a given scope. The
// shared dep-resolve / provider-call / decorator pipeline is encapsulated in
// the build function passed by the caller; each strategy decides whether
// (and where) to cache, and whether to call build at all.
type ScopeStrategy interface {
	Acquire(
		ctx context.Context,
		c *Container,
		key string,
		entry *ServiceEntry,
		build func() (any, error),
	) (any, error)
}

var scopeStrategies = map[scope.Scope]ScopeStrategy{
	scope.Singleton: singletonStrategy{},
	scope.Transient: transientStrategy{},
	scope.Request:   requestStrategy{},
	scope.Pooled:    pooledStrategy{},
}

func strategyFor(s scope.Scope) ScopeStrategy {
	if strategy, ok := scopeStrategies[s]; ok {
		return strategy
	}
	return singletonStrategy{}
}

// singletonStrategy gates the build with sync.Once and caches in entry.Instance.
// It also runs the lazy OnStart hook the first time a lazy entry is resolved
// after the container has started.
type singletonStrategy struct{}

func (singletonStrategy) Acquire(
	ctx context.Context,
	c *Container,
	key string,
	entry *ServiceEntry,
	build func() (any, error),
) (any, error) {
	if entry.Provider == nil {
		return entry.Instance, nil
	}

	entry.once.Do(func() {
		inst, err := build()
		if err != nil {
			entry.initErr = err
			return
		}
		c.registry.SetInstance(key, inst)
	})

	if entry.initErr != nil {
		return nil, entry.initErr
	}

	if entry.Lazy && !entry.StartRan && c.state == StateRunning {
		if err := c.runLazyStart(ctx, key, entry); err != nil {
			return nil, err
		}
	}

	return entry.Instance, nil
}

// transientStrategy never caches: every Resolve produces a fresh instance.
type transientStrategy struct{}

func (transientStrategy) Acquire(
	_ context.Context,
	_ *Container,
	_ string,
	_ *ServiceEntry,
	build func() (any, error),
) (any, error) {
	return build()
}

// requestStrategy caches in a per-context RequestScope. Errors if no
// RequestScope is attached to the context.
type requestStrategy struct{}

func (requestStrategy) Acquire(
	ctx context.Context,
	_ *Container,
	key string,
	_ *ServiceEntry,
	build func() (any, error),
) (any, error) {
	rs := getRequestScope(ctx)
	if rs == nil {
		return nil, fmt.Errorf("%w: %s; use WithRequestScope(ctx)", ErrRequestScopeMissing, key)
	}

	if instance, ok := rs.Get(key); ok {
		return instance, nil
	}

	instance, err := build()
	if err != nil {
		return nil, err
	}
	rs.Set(key, instance)
	return instance, nil
}

// pooledStrategy tries the entry's pool channel first; on miss, builds a fresh
// instance. The instance is not put into the pool here -- callers return
// instances via Container.Release once they're done with them.
type pooledStrategy struct{}

func (pooledStrategy) Acquire(
	_ context.Context,
	_ *Container,
	_ string,
	entry *ServiceEntry,
	build func() (any, error),
) (any, error) {
	if entry.pool != nil {
		select {
		case instance := <-entry.pool:
			return instance, nil
		default:
		}
	}
	return build()
}

// returnToPool tries to put an instance back into the entry's pool channel.
// Returns false if the entry isn't pooled or the channel is full.
func returnToPool(entry *ServiceEntry, instance any) bool {
	if entry.pool == nil {
		return false
	}
	select {
	case entry.pool <- instance:
		return true
	default:
		return false
	}
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
