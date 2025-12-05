package main

import (
	"context"
	"fmt"
	"time"

	"github.com/danpasecinic/needle"
)

type ExpensiveService struct {
	name string
}

func NewExpensiveService(name string) *ExpensiveService {
	fmt.Printf("[%s] Creating %s (expensive operation)...\n", time.Now().Format("15:04:05.000"), name)
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("[%s] %s created!\n", time.Now().Format("15:04:05.000"), name)
	return &ExpensiveService{name: name}
}

func (s *ExpensiveService) DoWork() {
	fmt.Printf("[%s] %s doing work\n", time.Now().Format("15:04:05.000"), s.name)
}

type EagerService struct {
	name string
}

func NewEagerService(name string) *EagerService {
	fmt.Printf("[%s] Creating %s...\n", time.Now().Format("15:04:05.000"), name)
	return &EagerService{name: name}
}

func (s *EagerService) DoWork() {
	fmt.Printf("[%s] %s doing work\n", time.Now().Format("15:04:05.000"), s.name)
}

func main() {
	c := needle.New()

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*EagerService, error) {
			return NewEagerService("EagerService"), nil
		},
		needle.WithOnStart(
			func(_ context.Context) error {
				fmt.Println("[lifecycle] EagerService OnStart hook running")
				return nil
			},
		),
		needle.WithOnStop(
			func(_ context.Context) error {
				fmt.Println("[lifecycle] EagerService OnStop hook running")
				return nil
			},
		),
	)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*ExpensiveService, error) {
			return NewExpensiveService("LazyExpensiveService"), nil
		},
		needle.WithLazy(),
		needle.WithOnStart(
			func(_ context.Context) error {
				fmt.Println("[lifecycle] LazyExpensiveService OnStart hook running")
				return nil
			},
		),
		needle.WithOnStop(
			func(_ context.Context) error {
				fmt.Println("[lifecycle] LazyExpensiveService OnStop hook running")
				return nil
			},
		),
	)

	fmt.Println("=== Starting container ===")
	ctx := context.Background()
	_ = c.Start(ctx)

	fmt.Println("\n=== Container started ===")
	fmt.Println("Notice: LazyExpensiveService was NOT created during startup!")

	fmt.Println("\n=== Using EagerService ===")
	eager := needle.MustInvoke[*EagerService](c)
	eager.DoWork()

	fmt.Println("\n=== First invoke of LazyExpensiveService ===")
	fmt.Println("Now the lazy service will be created...")
	lazy := needle.MustInvoke[*ExpensiveService](c)
	lazy.DoWork()

	fmt.Println("\n=== Second invoke of LazyExpensiveService ===")
	fmt.Println("Singleton - no recreation:")
	lazy2 := needle.MustInvoke[*ExpensiveService](c)
	lazy2.DoWork()

	fmt.Println("\n=== Stopping container ===")
	_ = c.Stop(ctx)

	fmt.Println("\n=== Container stopped ===")
}
