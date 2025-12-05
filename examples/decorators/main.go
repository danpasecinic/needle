package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/danpasecinic/needle"
)

type UserRepository interface {
	FindByID(ctx context.Context, id int) (string, error)
}

type PostgresUserRepository struct{}

func (r *PostgresUserRepository) FindByID(_ context.Context, id int) (string, error) {
	time.Sleep(10 * time.Millisecond)
	if id < 0 {
		return "", fmt.Errorf("invalid id: %d", id)
	}
	return fmt.Sprintf("User{id: %d, name: Alice}", id), nil
}

type LoggingRepository struct {
	inner  UserRepository
	logger *slog.Logger
}

func (r *LoggingRepository) FindByID(ctx context.Context, id int) (string, error) {
	r.logger.Info("repository call", "method", "FindByID", "id", id)
	result, err := r.inner.FindByID(ctx, id)
	if err != nil {
		r.logger.Error("repository error", "method", "FindByID", "error", err)
	}
	return result, err
}

type MetricsRepository struct {
	inner UserRepository
}

func (r *MetricsRepository) FindByID(ctx context.Context, id int) (string, error) {
	start := time.Now()
	result, err := r.inner.FindByID(ctx, id)
	duration := time.Since(start)
	fmt.Printf("[metrics] FindByID took %v\n", duration)
	return result, err
}

type CachingRepository struct {
	inner UserRepository
	cache map[int]string
}

func (r *CachingRepository) FindByID(ctx context.Context, id int) (string, error) {
	if cached, ok := r.cache[id]; ok {
		fmt.Printf("[cache] HIT for id=%d\n", id)
		return cached, nil
	}
	fmt.Printf("[cache] MISS for id=%d\n", id)
	result, err := r.inner.FindByID(ctx, id)
	if err == nil {
		r.cache[id] = result
	}
	return result, err
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := needle.New()

	_ = needle.ProvideValue(c, logger)

	_ = needle.Provide(
		c, func(_ context.Context, _ needle.Resolver) (*PostgresUserRepository, error) {
			return &PostgresUserRepository{}, nil
		},
	)

	_ = needle.Bind[UserRepository, *PostgresUserRepository](c)

	needle.Decorate(
		c, func(_ context.Context, _ needle.Resolver, repo UserRepository) (UserRepository, error) {
			log := needle.MustInvoke[*slog.Logger](c)
			fmt.Println("Applying logging decorator...")
			return &LoggingRepository{inner: repo, logger: log}, nil
		},
	)

	needle.Decorate(
		c, func(_ context.Context, _ needle.Resolver, repo UserRepository) (UserRepository, error) {
			fmt.Println("Applying metrics decorator...")
			return &MetricsRepository{inner: repo}, nil
		},
	)

	needle.Decorate(
		c, func(_ context.Context, _ needle.Resolver, repo UserRepository) (UserRepository, error) {
			fmt.Println("Applying caching decorator...")
			return &CachingRepository{inner: repo, cache: make(map[int]string)}, nil
		},
	)

	fmt.Println("=== Resolving UserRepository ===")
	repo := needle.MustInvoke[UserRepository](c)

	fmt.Println("\n=== First call (cache miss) ===")
	ctx := context.Background()
	result, _ := repo.FindByID(ctx, 42)
	fmt.Printf("Result: %s\n", result)

	fmt.Println("\n=== Second call (cache hit) ===")
	result, _ = repo.FindByID(ctx, 42)
	fmt.Printf("Result: %s\n", result)

	fmt.Println("\n=== Third call (different ID, cache miss) ===")
	result, _ = repo.FindByID(ctx, 99)
	fmt.Printf("Result: %s\n", result)
}
