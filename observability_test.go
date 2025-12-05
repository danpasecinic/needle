package needle_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danpasecinic/needle"
)

type HealthyService struct{}

func (s *HealthyService) HealthCheck(ctx context.Context) error {
	return nil
}

type UnhealthyService struct{}

func (s *UnhealthyService) HealthCheck(ctx context.Context) error {
	return errors.New("service unhealthy")
}

type ReadyService struct{}

func (s *ReadyService) ReadinessCheck(ctx context.Context) error {
	return nil
}

type NotReadyService struct{}

func (s *NotReadyService) ReadinessCheck(ctx context.Context) error {
	return errors.New("service not ready")
}

func TestHealthCheckHealthyService(t *testing.T) {
	t.Parallel()

	c := needle.New()
	ctx := context.Background()

	err := needle.ProvideValue(c, &HealthyService{})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	_ = c.Start(ctx)

	err = c.Live(ctx)
	if err != nil {
		t.Errorf("Live() should pass for healthy service: %v", err)
	}

	reports := c.Health(ctx)
	if len(reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(reports))
	}

	if reports[0].Status != needle.HealthStatusUp {
		t.Errorf("expected status up, got %s", reports[0].Status)
	}
}

func TestHealthCheckUnhealthyService(t *testing.T) {
	t.Parallel()

	c := needle.New()
	ctx := context.Background()

	err := needle.ProvideValue(c, &UnhealthyService{})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	_ = c.Start(ctx)

	err = c.Live(ctx)
	if err == nil {
		t.Error("Live() should fail for unhealthy service")
	}

	if !needle.IsHealthCheckFailed(err) {
		t.Error("expected health check failed error")
	}

	reports := c.Health(ctx)
	if len(reports) != 1 {
		t.Errorf("expected 1 report, got %d", len(reports))
	}

	if reports[0].Status != needle.HealthStatusDown {
		t.Errorf("expected status down, got %s", reports[0].Status)
	}
}

func TestReadinessCheckReadyService(t *testing.T) {
	t.Parallel()

	c := needle.New()
	ctx := context.Background()

	err := needle.ProvideValue(c, &ReadyService{})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	_ = c.Start(ctx)

	err = c.Ready(ctx)
	if err != nil {
		t.Errorf("Ready() should pass for ready service: %v", err)
	}
}

func TestReadinessCheckNotReadyService(t *testing.T) {
	t.Parallel()

	c := needle.New()
	ctx := context.Background()

	err := needle.ProvideValue(c, &NotReadyService{})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	_ = c.Start(ctx)

	err = c.Ready(ctx)
	if err == nil {
		t.Error("Ready() should fail for not ready service")
	}
}

func TestHealthCheckNoHealthCheckers(t *testing.T) {
	t.Parallel()

	c := needle.New()
	ctx := context.Background()

	err := needle.ProvideValue(c, &Config{Port: 8080})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	_ = c.Start(ctx)

	err = c.Live(ctx)
	if err != nil {
		t.Errorf("Live() should pass when no health checkers: %v", err)
	}

	reports := c.Health(ctx)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports, got %d", len(reports))
	}
}

func TestResolveObserver(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	var lastKey string
	var lastErr error

	c := needle.New(
		needle.WithResolveObserver(func(key string, duration time.Duration, err error) {
			callCount.Add(1)
			lastKey = key
			lastErr = err
		}),
	)

	err := needle.ProvideValue(c, &Config{Port: 8080})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	_, err = needle.Invoke[*Config](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if callCount.Load() != 1 {
		t.Errorf("expected 1 resolve hook call, got %d", callCount.Load())
	}

	if lastKey == "" {
		t.Error("expected key to be set")
	}

	if lastErr != nil {
		t.Errorf("expected no error, got %v", lastErr)
	}
}

func TestResolveObserverOnError(t *testing.T) {
	t.Parallel()

	var lastErr error

	c := needle.New(
		needle.WithResolveObserver(func(key string, duration time.Duration, err error) {
			lastErr = err
		}),
	)

	_, _ = needle.Invoke[*Config](c)

	if lastErr == nil {
		t.Error("expected error to be passed to hook")
	}
}

func TestProvideObserver(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	var keys []string

	c := needle.New(
		needle.WithProvideObserver(func(key string) {
			callCount.Add(1)
			keys = append(keys, key)
		}),
	)

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	_ = needle.ProvideValue(c, &Database{Name: "test"})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 provide hook calls, got %d", callCount.Load())
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestStartObserver(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	var keys []string

	c := needle.New(
		needle.WithStartObserver(func(key string, duration time.Duration, err error) {
			callCount.Add(1)
			keys = append(keys, key)
		}),
	)

	_ = needle.ProvideValue(c, &Config{Port: 8080})

	ctx := context.Background()
	err := c.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if callCount.Load() != 1 {
		t.Errorf("expected 1 start hook call, got %d", callCount.Load())
	}
}

func TestStopObserver(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	c := needle.New(
		needle.WithStopObserver(func(key string, duration time.Duration, err error) {
			callCount.Add(1)
		}),
	)

	_ = needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
		return &Server{}, nil
	}, needle.WithOnStop(func(ctx context.Context) error {
		return nil
	}))

	ctx := context.Background()
	_ = c.Start(ctx)
	_ = c.Stop(ctx)

	if callCount.Load() != 1 {
		t.Errorf("expected 1 stop hook call, got %d", callCount.Load())
	}
}

type SlowHealthService struct{}

func (s *SlowHealthService) HealthCheck(ctx context.Context) error {
	time.Sleep(time.Millisecond)
	return nil
}

func TestHealthReportLatency(t *testing.T) {
	t.Parallel()

	c := needle.New()
	ctx := context.Background()

	_ = needle.ProvideValue(c, &SlowHealthService{})
	_ = c.Start(ctx)

	reports := c.Health(ctx)

	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	if reports[0].Latency < time.Millisecond {
		t.Errorf("expected latency >= 1ms, got %v", reports[0].Latency)
	}
}

func TestMultipleObservers(t *testing.T) {
	t.Parallel()

	var count1, count2 atomic.Int32

	c := needle.New(
		needle.WithResolveObserver(func(key string, duration time.Duration, err error) {
			count1.Add(1)
		}),
		needle.WithResolveObserver(func(key string, duration time.Duration, err error) {
			count2.Add(1)
		}),
	)

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	_, _ = needle.Invoke[*Config](c)

	if count1.Load() != 1 || count2.Load() != 1 {
		t.Errorf("expected both observers to be called, got %d and %d", count1.Load(), count2.Load())
	}
}
