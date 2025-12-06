package main

import (
	"context"
	"fmt"
	"time"

	"github.com/danpasecinic/needle"
)

type Config struct {
	Value string
}

type DatabaseA struct {
	name string
}

type DatabaseB struct {
	name string
}

type CacheA struct {
	name string
}

type CacheB struct {
	name string
}

type ServiceA struct {
	db    *DatabaseA
	cache *CacheA
}

type ServiceB struct {
	db    *DatabaseB
	cache *CacheB
}

type Gateway struct {
	svcA *ServiceA
	svcB *ServiceB
}

func main() {
	fmt.Println("=== Sequential Startup ===")
	runSequential()

	fmt.Println("\n=== Parallel Startup ===")
	runParallel()
}

func runSequential() {
	c := needle.New()
	registerProviders(c)

	start := time.Now()
	ctx := context.Background()
	_ = c.Start(ctx)
	fmt.Printf("Startup took: %v\n", time.Since(start))

	_ = c.Stop(ctx)
}

func runParallel() {
	c := needle.New(needle.WithParallel())
	registerProviders(c)

	start := time.Now()
	ctx := context.Background()
	_ = c.Start(ctx)
	fmt.Printf("Startup took: %v\n", time.Since(start))

	fmt.Println("\nDependency graph:")
	c.PrintGraph()

	_ = c.Stop(ctx)
}

func registerProviders(c *needle.Container) {
	_ = needle.ProvideValue(c, &Config{Value: "config"})

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*DatabaseA, error) {
			fmt.Printf("[%s] Starting DatabaseA...\n", timestamp())
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("[%s] DatabaseA ready\n", timestamp())
			return &DatabaseA{name: "db-a"}, nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*DatabaseB, error) {
			fmt.Printf("[%s] Starting DatabaseB...\n", timestamp())
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("[%s] DatabaseB ready\n", timestamp())
			return &DatabaseB{name: "db-b"}, nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*CacheA, error) {
			fmt.Printf("[%s] Starting CacheA...\n", timestamp())
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("[%s] CacheA ready\n", timestamp())
			return &CacheA{name: "cache-a"}, nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*CacheB, error) {
			fmt.Printf("[%s] Starting CacheB...\n", timestamp())
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("[%s] CacheB ready\n", timestamp())
			return &CacheB{name: "cache-b"}, nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*ServiceA, error) {
			db := needle.MustInvoke[*DatabaseA](c)
			cache := needle.MustInvoke[*CacheA](c)
			fmt.Printf("[%s] Starting ServiceA...\n", timestamp())
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("[%s] ServiceA ready\n", timestamp())
			return &ServiceA{db: db, cache: cache}, nil
		},
		needle.WithDependencies("*main.DatabaseA", "*main.CacheA"),
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*ServiceB, error) {
			db := needle.MustInvoke[*DatabaseB](c)
			cache := needle.MustInvoke[*CacheB](c)
			fmt.Printf("[%s] Starting ServiceB...\n", timestamp())
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("[%s] ServiceB ready\n", timestamp())
			return &ServiceB{db: db, cache: cache}, nil
		},
		needle.WithDependencies("*main.DatabaseB", "*main.CacheB"),
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*Gateway, error) {
			svcA := needle.MustInvoke[*ServiceA](c)
			svcB := needle.MustInvoke[*ServiceB](c)
			fmt.Printf("[%s] Starting Gateway...\n", timestamp())
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("[%s] Gateway ready\n", timestamp())
			return &Gateway{svcA: svcA, svcB: svcB}, nil
		},
		needle.WithDependencies("*main.ServiceA", "*main.ServiceB"),
	)
}

var startTime = time.Now()

func timestamp() string {
	return fmt.Sprintf("%6.0fms", float64(time.Since(startTime).Milliseconds()))
}
