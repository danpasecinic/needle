package container

import (
	"context"
	"sync"

	"github.com/danpasecinic/needle/internal/scope"
)

type ProviderFunc func(ctx context.Context, r Resolver) (any, error)

type Resolver interface {
	Resolve(ctx context.Context, key string) (any, error)
	Has(key string) bool
}

type Hook func(ctx context.Context) error

type ServiceEntry struct {
	Key          string
	Provider     ProviderFunc
	Instance     any
	Instantiated bool
	Dependencies []string
	OnStart      []Hook
	OnStop       []Hook
	Scope        scope.Scope
	PoolSize     int
	pool         chan any
	Lazy         bool
	StartRan     bool
}

type Registry struct {
	mu       sync.RWMutex
	services map[string]*ServiceEntry
}

func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]*ServiceEntry),
	}
}

func (r *Registry) Register(key string, provider ProviderFunc, dependencies []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[key] = &ServiceEntry{
		Key:          key,
		Provider:     provider,
		Dependencies: dependencies,
	}
	return nil
}

func (r *Registry) RegisterUnsafe(key string, provider ProviderFunc, dependencies []string) {
	r.services[key] = &ServiceEntry{
		Key:          key,
		Provider:     provider,
		Dependencies: dependencies,
	}
}

func (r *Registry) RegisterValue(key string, value any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[key] = &ServiceEntry{
		Key:          key,
		Instance:     value,
		Instantiated: true,
	}
	return nil
}

func (r *Registry) RegisterValueUnsafe(key string, value any) {
	r.services[key] = &ServiceEntry{
		Key:          key,
		Instance:     value,
		Instantiated: true,
	}
}

func (r *Registry) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.services[key]
	return exists
}

func (r *Registry) HasUnsafe(key string) bool {
	_, exists := r.services[key]
	return exists
}

func (r *Registry) Get(key string) (*ServiceEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[key]
	return entry, exists
}

func (r *Registry) GetInstance(key string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[key]
	if !exists || !entry.Instantiated {
		return nil, false
	}
	return entry.Instance, true
}

func (r *Registry) GetInstanceFast(key string) (any, bool) {
	r.mu.RLock()
	entry, exists := r.services[key]
	if !exists {
		r.mu.RUnlock()
		return nil, false
	}
	if entry.Instantiated && entry.Scope == scope.Singleton {
		instance := entry.Instance
		r.mu.RUnlock()
		return instance, true
	}
	r.mu.RUnlock()
	return nil, false
}

func (r *Registry) SetInstance(key string, instance any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.Instance = instance
		entry.Instantiated = true
	}
}

func (r *Registry) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.services))
	for key := range r.services {
		keys = append(keys, key)
	}
	return keys
}

func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.services)
}

func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services = make(map[string]*ServiceEntry)
}

func (r *Registry) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, key)
}

func (r *Registry) RemoveUnsafe(key string) {
	delete(r.services, key)
}

func (r *Registry) Dependencies(key string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[key]
	if !exists {
		return nil
	}

	deps := make([]string, len(entry.Dependencies))
	copy(deps, entry.Dependencies)
	return deps
}

func (r *Registry) AllDependencies() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deps := make(map[string][]string, len(r.services))
	for key, entry := range r.services {
		d := make([]string, len(entry.Dependencies))
		copy(d, entry.Dependencies)
		deps[key] = d
	}
	return deps
}

func (r *Registry) AddOnStart(key string, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.OnStart = append(entry.OnStart, hook)
	}
}

func (r *Registry) AddOnStop(key string, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.OnStop = append(entry.OnStop, hook)
	}
}

func (r *Registry) GetEntry(key string) (*ServiceEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[key]
	return entry, exists
}

func (r *Registry) AllEntries() []*ServiceEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]*ServiceEntry, 0, len(r.services))
	for _, entry := range r.services {
		entries = append(entries, entry)
	}
	return entries
}

func (r *Registry) SetScope(key string, s scope.Scope) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.Scope = s
	}
}

func (r *Registry) SetPoolSize(key string, size int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.PoolSize = size
		if size > 0 {
			entry.pool = make(chan any, size)
		}
	}
}

func (r *Registry) AcquireFromPool(key string) (any, bool) {
	r.mu.RLock()
	entry, exists := r.services[key]
	r.mu.RUnlock()

	if !exists || entry.pool == nil {
		return nil, false
	}

	select {
	case instance := <-entry.pool:
		return instance, true
	default:
		return nil, false
	}
}

func (r *Registry) ReleaseToPool(key string, instance any) bool {
	r.mu.RLock()
	entry, exists := r.services[key]
	r.mu.RUnlock()

	if !exists || entry.pool == nil {
		return false
	}

	select {
	case entry.pool <- instance:
		return true
	default:
		return false
	}
}

func (r *Registry) SetLazy(key string, lazy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.Lazy = lazy
	}
}

func (r *Registry) IsLazy(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if entry, exists := r.services[key]; exists {
		return entry.Lazy
	}
	return false
}

func (r *Registry) SetStartRan(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[key]; exists {
		entry.StartRan = true
	}
}
