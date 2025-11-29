package container

import (
	"context"
	"sync"
)

type ProviderFunc func(ctx context.Context, r Resolver) (any, error)

type Resolver interface {
	Resolve(ctx context.Context, key string) (any, error)
	Has(key string) bool
}

type ServiceEntry struct {
	Key          string
	Provider     ProviderFunc
	Instance     any
	Instantiated bool
	Dependencies []string
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

func (r *Registry) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
