package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/danpasecinic/needle"
)

type Database struct {
	healthy bool
	ready   bool
}

func NewDatabase() *Database {
	return &Database{healthy: true, ready: true}
}

func (d *Database) HealthCheck(_ context.Context) error {
	if !d.healthy {
		return errors.New("database connection lost")
	}
	return nil
}

func (d *Database) ReadinessCheck(_ context.Context) error {
	if !d.ready {
		return errors.New("database not ready for queries")
	}
	return nil
}

func (d *Database) SetHealthy(healthy bool) {
	d.healthy = healthy
}

func (d *Database) SetReady(ready bool) {
	d.ready = ready
}

type Cache struct {
	healthy bool
}

func NewCache() *Cache {
	return &Cache{healthy: true}
}

func (c *Cache) HealthCheck(_ context.Context) error {
	if !c.healthy {
		return errors.New("cache unavailable")
	}
	return nil
}

func (c *Cache) SetHealthy(healthy bool) {
	c.healthy = healthy
}

type MessageQueue struct {
	ready bool
}

func NewMessageQueue() *MessageQueue {
	return &MessageQueue{ready: true}
}

func (m *MessageQueue) ReadinessCheck(_ context.Context) error {
	if !m.ready {
		return errors.New("message queue not ready")
	}
	return nil
}

func (m *MessageQueue) SetReady(ready bool) {
	m.ready = ready
}

func main() {
	c := needle.New()

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*Database, error) {
			return NewDatabase(), nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*Cache, error) {
			return NewCache(), nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*MessageQueue, error) {
			return NewMessageQueue(), nil
		},
	)

	ctx := context.Background()
	_ = c.Start(ctx)

	db := needle.MustInvoke[*Database](c)
	cache := needle.MustInvoke[*Cache](c)
	mq := needle.MustInvoke[*MessageQueue](c)

	fmt.Println("=== All services healthy ===")
	printHealthStatus(c, ctx)

	fmt.Println("\n=== Database becomes unhealthy ===")
	db.SetHealthy(false)
	printHealthStatus(c, ctx)

	fmt.Println("\n=== Database healthy again, but not ready ===")
	db.SetHealthy(true)
	db.SetReady(false)
	printHealthStatus(c, ctx)

	fmt.Println("\n=== Cache unhealthy ===")
	db.SetReady(true)
	cache.SetHealthy(false)
	printHealthStatus(c, ctx)

	fmt.Println("\n=== Message queue not ready ===")
	cache.SetHealthy(true)
	mq.SetReady(false)
	printHealthStatus(c, ctx)

	fmt.Println("\n=== Detailed health reports ===")
	mq.SetReady(true)
	reports := c.Health(ctx)
	for _, r := range reports {
		fmt.Printf("  %s: status=%s, latency=%v\n", r.Name, r.Status, r.Latency)
	}

	fmt.Println("\n=== HTTP health endpoint example ===")
	fmt.Println("In a real app, you'd expose these as HTTP endpoints:")
	fmt.Println(
		`
http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    if err := c.Live(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte(err.Error()))
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})

http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
    if err := c.Ready(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte(err.Error()))
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})`,
	)

	_ = c.Stop(ctx)
}

func printHealthStatus(c *needle.Container, ctx context.Context) {
	if err := c.Live(ctx); err != nil {
		fmt.Printf("  Liveness:  FAIL - %v\n", err)
	} else {
		fmt.Println("  Liveness:  PASS")
	}

	if err := c.Ready(ctx); err != nil {
		fmt.Printf("  Readiness: FAIL - %v\n", err)
	} else {
		fmt.Println("  Readiness: PASS")
	}
}
