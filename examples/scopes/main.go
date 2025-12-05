package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/danpasecinic/needle"
)

type SingletonCounter struct {
	id int64
}

var singletonID atomic.Int64

func NewSingletonCounter() *SingletonCounter {
	return &SingletonCounter{id: singletonID.Add(1)}
}

type TransientCounter struct {
	id int64
}

var transientID atomic.Int64

func NewTransientCounter() *TransientCounter {
	return &TransientCounter{id: transientID.Add(1)}
}

type RequestCounter struct {
	id int64
}

var requestID atomic.Int64

func NewRequestCounter() *RequestCounter {
	return &RequestCounter{id: requestID.Add(1)}
}

type PooledConnection struct {
	id int64
}

var pooledID atomic.Int64

func NewPooledConnection() *PooledConnection {
	return &PooledConnection{id: pooledID.Add(1)}
}

func main() {
	c := needle.New()

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*SingletonCounter, error) {
			return NewSingletonCounter(), nil
		},
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*TransientCounter, error) {
			return NewTransientCounter(), nil
		},
		needle.WithScope(needle.Transient),
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*RequestCounter, error) {
			return NewRequestCounter(), nil
		},
		needle.WithScope(needle.Request),
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*PooledConnection, error) {
			return NewPooledConnection(), nil
		},
		needle.WithPoolSize(3),
	)

	fmt.Println("=== Singleton Scope ===")
	fmt.Println("Same instance returned every time:")
	for i := 0; i < 3; i++ {
		s := needle.MustInvoke[*SingletonCounter](c)
		fmt.Printf("  Invoke %d: instance ID = %d\n", i+1, s.id)
	}

	fmt.Println("\n=== Transient Scope ===")
	fmt.Println("New instance returned every time:")
	for i := 0; i < 3; i++ {
		t := needle.MustInvoke[*TransientCounter](c)
		fmt.Printf("  Invoke %d: instance ID = %d\n", i+1, t.id)
	}

	fmt.Println("\n=== Request Scope ===")
	fmt.Println("Same instance within a request, different across requests:")

	for req := 1; req <= 2; req++ {
		ctx := needle.WithRequestScope(context.Background())
		fmt.Printf("  Request %d:\n", req)
		for i := 0; i < 3; i++ {
			r, _ := needle.InvokeCtx[*RequestCounter](ctx, c)
			fmt.Printf("    Invoke %d: instance ID = %d\n", i+1, r.id)
		}
	}

	fmt.Println("\n=== Pooled Scope ===")
	fmt.Println("Reuses instances from a pool:")

	var connections []*PooledConnection
	for i := 0; i < 3; i++ {
		conn := needle.MustInvoke[*PooledConnection](c)
		connections = append(connections, conn)
		fmt.Printf("  Acquired: instance ID = %d\n", conn.id)
	}

	fmt.Println("  Releasing all connections back to pool...")
	for _, conn := range connections {
		c.Release("*main.PooledConnection", conn)
	}

	fmt.Println("  Acquiring again (should reuse from pool):")
	for i := 0; i < 3; i++ {
		conn := needle.MustInvoke[*PooledConnection](c)
		fmt.Printf("  Acquired: instance ID = %d\n", conn.id)
	}
}
