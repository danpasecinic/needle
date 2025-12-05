package main

import (
	"context"
	"fmt"

	"github.com/danpasecinic/needle"
)

type Cache interface {
	Get(key string) (string, bool)
	Set(key string, value string)
}

type RedisCache struct {
	data map[string]string
}

func NewRedisCache() *RedisCache {
	fmt.Println("[redis] Creating Redis cache")
	return &RedisCache{data: make(map[string]string)}
}

func (c *RedisCache) Get(key string) (string, bool) {
	v, ok := c.data[key]
	return v, ok
}

func (c *RedisCache) Set(key string, value string) {
	c.data[key] = value
}

type InMemoryCache struct {
	data map[string]string
}

func NewInMemoryCache() *InMemoryCache {
	fmt.Println("[memory] Creating in-memory cache (fallback)")
	return &InMemoryCache{data: make(map[string]string)}
}

func (c *InMemoryCache) Get(key string) (string, bool) {
	v, ok := c.data[key]
	return v, ok
}

func (c *InMemoryCache) Set(key string, value string) {
	c.data[key] = value
}

type Metrics interface {
	Inc(name string)
}

type PrometheusMetrics struct{}

func (m *PrometheusMetrics) Inc(name string) {
	fmt.Printf("[prometheus] %s++\n", name)
}

type NoOpMetrics struct{}

func (m *NoOpMetrics) Inc(_ string) {}

type UserService struct {
	cache   Cache
	metrics Metrics
}

func (s *UserService) GetUser(id int) string {
	s.metrics.Inc("user_requests")

	key := fmt.Sprintf("user:%d", id)
	if cached, ok := s.cache.Get(key); ok {
		s.metrics.Inc("cache_hits")
		return cached
	}

	s.metrics.Inc("cache_misses")
	user := fmt.Sprintf("User{id: %d}", id)
	s.cache.Set(key, user)
	return user
}

func main() {
	fmt.Println("=== Scenario 1: All dependencies available ===")
	runWithAllDeps()

	fmt.Println("\n=== Scenario 2: No cache, no metrics ===")
	runWithoutOptionalDeps()

	fmt.Println("\n=== Scenario 3: Using Optional API directly ===")
	demonstrateOptionalAPI()
}

func runWithAllDeps() {
	c := needle.New()

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*RedisCache, error) {
			return NewRedisCache(), nil
		},
	)
	_ = needle.Bind[Cache, *RedisCache](c)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*PrometheusMetrics, error) {
			return &PrometheusMetrics{}, nil
		},
	)
	_ = needle.Bind[Metrics, *PrometheusMetrics](c)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*UserService, error) {
			cache := needle.InvokeOptional[Cache](c).OrElseFunc(
				func() Cache {
					return NewInMemoryCache()
				},
			)
			metrics := needle.InvokeOptional[Metrics](c).OrElse(&NoOpMetrics{})
			return &UserService{cache: cache, metrics: metrics}, nil
		},
	)

	svc := needle.MustInvoke[*UserService](c)
	fmt.Println(svc.GetUser(42))
	fmt.Println(svc.GetUser(42))
}

func runWithoutOptionalDeps() {
	c := needle.New()

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*UserService, error) {
			cache := needle.InvokeOptional[Cache](c).OrElseFunc(
				func() Cache {
					return NewInMemoryCache()
				},
			)
			metrics := needle.InvokeOptional[Metrics](c).OrElse(&NoOpMetrics{})
			return &UserService{cache: cache, metrics: metrics}, nil
		},
	)

	svc := needle.MustInvoke[*UserService](c)
	fmt.Println(svc.GetUser(42))
	fmt.Println(svc.GetUser(42))
}

func demonstrateOptionalAPI() {
	c := needle.New()

	_ = needle.ProvideValue(c, &RedisCache{data: make(map[string]string)})
	_ = needle.Bind[Cache, *RedisCache](c)

	fmt.Println("--- Present() and Value() ---")
	opt := needle.InvokeOptional[Cache](c)
	if opt.Present() {
		cache := opt.Value()
		cache.Set("foo", "bar")
		fmt.Printf("Cache present, set foo=bar\n")
	}

	fmt.Println("\n--- Get() with boolean ---")
	if cache, ok := opt.Get(); ok {
		v, _ := cache.Get("foo")
		fmt.Printf("Got value: %s\n", v)
	}

	fmt.Println("\n--- OrElse() ---")
	cache := needle.InvokeOptional[Cache](c).OrElse(NewInMemoryCache())
	fmt.Printf("Cache type: %T\n", cache)

	fmt.Println("\n--- OrElseFunc() (lazy) ---")
	cache = needle.InvokeOptional[Cache](c).OrElseFunc(
		func() Cache {
			fmt.Println("This won't print because cache exists")
			return NewInMemoryCache()
		},
	)
	fmt.Printf("Cache type: %T\n", cache)

	fmt.Println("\n--- Missing dependency ---")
	optMetrics := needle.InvokeOptional[Metrics](c)
	fmt.Printf("Metrics present: %v\n", optMetrics.Present())
	metrics := optMetrics.OrElse(&NoOpMetrics{})
	fmt.Printf("Metrics type: %T\n", metrics)

	fmt.Println("\n--- Named optional ---")
	optNamed := needle.InvokeOptionalNamed[Cache](c, "session")
	fmt.Printf("Named cache present: %v\n", optNamed.Present())
}
