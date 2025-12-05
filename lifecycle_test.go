package needle

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danpasecinic/needle/internal/reflect"
)

func TestContainer_StartStop(t *testing.T) {
	t.Parallel()

	c := New()

	var startCount, stopCount atomic.Int32

	err := Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "test"}, nil
		},
		WithOnStart(
			func(ctx context.Context) error {
				startCount.Add(1)
				return nil
			},
		),
		WithOnStop(
			func(ctx context.Context) error {
				stopCount.Add(1)
				return nil
			},
		),
	)
	if err != nil {
		t.Fatalf("failed to provide: %v", err)
	}

	ctx := context.Background()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	if startCount.Load() != 1 {
		t.Errorf("expected start count 1, got %d", startCount.Load())
	}

	if err := c.Stop(ctx); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	if stopCount.Load() != 1 {
		t.Errorf("expected stop count 1, got %d", stopCount.Load())
	}
}

func TestContainer_StartOrder(t *testing.T) {
	t.Parallel()

	c := New()

	var order []string

	_ = ProvideValue(
		c, &testConfig{value: "config"},
		WithOnStart(
			func(ctx context.Context) error {
				order = append(order, "config")
				return nil
			},
		),
	)

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testDatabase, error) {
			_ = MustInvoke[*testConfig](c)
			return &testDatabase{}, nil
		},
		WithDependencies(reflect.TypeKey[*testConfig]()),
		WithOnStart(
			func(ctx context.Context) error {
				order = append(order, "database")
				return nil
			},
		),
	)

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testServer, error) {
			_ = MustInvoke[*testDatabase](c)
			return &testServer{}, nil
		},
		WithDependencies(reflect.TypeKey[*testDatabase]()),
		WithOnStart(
			func(ctx context.Context) error {
				order = append(order, "server")
				return nil
			},
		),
	)

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	expected := []string{"config", "database", "server"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected order[%d] = %s, got %s", i, v, order[i])
		}
	}

	_ = c.Stop(ctx)
}

func TestContainer_StopOrder(t *testing.T) {
	t.Parallel()

	c := New()

	var order []string

	_ = ProvideValue(
		c, &testConfig{value: "config"},
		WithOnStop(
			func(ctx context.Context) error {
				order = append(order, "config")
				return nil
			},
		),
	)

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testDatabase, error) {
			_ = MustInvoke[*testConfig](c)
			return &testDatabase{}, nil
		},
		WithDependencies(reflect.TypeKey[*testConfig]()),
		WithOnStop(
			func(ctx context.Context) error {
				order = append(order, "database")
				return nil
			},
		),
	)

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testServer, error) {
			_ = MustInvoke[*testDatabase](c)
			return &testServer{}, nil
		},
		WithDependencies(reflect.TypeKey[*testDatabase]()),
		WithOnStop(
			func(ctx context.Context) error {
				order = append(order, "server")
				return nil
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)

	if err := c.Stop(ctx); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	expected := []string{"server", "database", "config"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected order[%d] = %s, got %s", i, v, order[i])
		}
	}
}

func TestContainer_StartError(t *testing.T) {
	t.Parallel()

	c := New()

	expectedErr := errors.New("start failed")

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "test"}, nil
		},
		WithOnStart(
			func(ctx context.Context) error {
				return expectedErr
			},
		),
	)

	ctx := context.Background()
	err := c.Start(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}
}

func TestContainer_StopError(t *testing.T) {
	t.Parallel()

	c := New()

	expectedErr := errors.New("stop failed")

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "test"}, nil
		},
		WithOnStop(
			func(ctx context.Context) error {
				return expectedErr
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)

	err := c.Stop(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestContainer_MultipleHooks(t *testing.T) {
	t.Parallel()

	c := New()

	var order []string

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "test"}, nil
		},
		WithOnStart(
			func(ctx context.Context) error {
				order = append(order, "start1")
				return nil
			},
		),
		WithOnStart(
			func(ctx context.Context) error {
				order = append(order, "start2")
				return nil
			},
		),
		WithOnStop(
			func(ctx context.Context) error {
				order = append(order, "stop1")
				return nil
			},
		),
		WithOnStop(
			func(ctx context.Context) error {
				order = append(order, "stop2")
				return nil
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)
	_ = c.Stop(ctx)

	expected := []string{"start1", "start2", "stop2", "stop1"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected order[%d] = %s, got %s", i, v, order[i])
		}
	}
}

func TestContainer_Run(t *testing.T) {
	t.Parallel()

	c := New()

	var started, stopped atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "test"}, nil
		},
		WithOnStart(
			func(ctx context.Context) error {
				started.Store(true)
				return nil
			},
		),
		WithOnStop(
			func(ctx context.Context) error {
				stopped.Store(true)
				return nil
			},
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := c.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !started.Load() {
		t.Error("expected service to be started")
	}
	if !stopped.Load() {
		t.Error("expected service to be stopped")
	}
}

func TestContainer_DoubleStart(t *testing.T) {
	t.Parallel()

	c := New()

	_ = ProvideValue(c, &testConfig{value: "config"})

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	err := c.Start(ctx)
	if err == nil {
		t.Error("expected error on double start")
	}

	_ = c.Stop(ctx)
}

func TestContainer_StopWithoutStart(t *testing.T) {
	t.Parallel()

	c := New()

	_ = ProvideValue(c, &testConfig{value: "config"})

	ctx := context.Background()
	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("expected no error on stop without start, got %v", err)
	}
}

type testConfig struct {
	value string
}

type testDatabase struct{}

type testServer struct{}

type testService struct {
	name string
}

func TestContainer_LazyProvider(t *testing.T) {
	t.Parallel()

	c := New()

	var instantiated, started atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			instantiated.Store(true)
			return &testService{name: "lazy"}, nil
		},
		WithLazy(),
		WithOnStart(
			func(ctx context.Context) error {
				started.Store(true)
				return nil
			},
		),
	)

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	if instantiated.Load() {
		t.Error("lazy service should not be instantiated during Start")
	}
	if started.Load() {
		t.Error("lazy service OnStart should not run during Start")
	}

	_, err := Invoke[*testService](c)
	if err != nil {
		t.Fatalf("failed to invoke: %v", err)
	}

	if !instantiated.Load() {
		t.Error("lazy service should be instantiated after Invoke")
	}
	if !started.Load() {
		t.Error("lazy service OnStart should run after first Invoke")
	}

	_ = c.Stop(ctx)
}

func TestContainer_LazyProviderOnStartRunsOnce(t *testing.T) {
	t.Parallel()

	c := New()

	var startCount atomic.Int32

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "lazy"}, nil
		},
		WithLazy(),
		WithOnStart(
			func(ctx context.Context) error {
				startCount.Add(1)
				return nil
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)

	_, _ = Invoke[*testService](c)
	_, _ = Invoke[*testService](c)
	_, _ = Invoke[*testService](c)

	if startCount.Load() != 1 {
		t.Errorf("expected OnStart to run once, ran %d times", startCount.Load())
	}

	_ = c.Stop(ctx)
}

func TestContainer_LazyProviderStopHook(t *testing.T) {
	t.Parallel()

	c := New()

	var stopped atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "lazy"}, nil
		},
		WithLazy(),
		WithOnStop(
			func(ctx context.Context) error {
				stopped.Store(true)
				return nil
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)
	_, _ = Invoke[*testService](c)
	_ = c.Stop(ctx)

	if !stopped.Load() {
		t.Error("lazy service OnStop should run during Stop")
	}
}

func TestContainer_LazyProviderNotInstantiatedNoStop(t *testing.T) {
	t.Parallel()

	c := New()

	var stopped atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "lazy"}, nil
		},
		WithLazy(),
		WithOnStop(
			func(ctx context.Context) error {
				stopped.Store(true)
				return nil
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)
	_ = c.Stop(ctx)

	if stopped.Load() {
		t.Error("lazy service OnStop should not run if never instantiated")
	}
}

func TestContainer_LazyProviderBeforeStart(t *testing.T) {
	t.Parallel()

	c := New()

	var started atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "lazy"}, nil
		},
		WithLazy(),
		WithOnStart(
			func(ctx context.Context) error {
				started.Store(true)
				return nil
			},
		),
	)

	_, err := Invoke[*testService](c)
	if err != nil {
		t.Fatalf("failed to invoke: %v", err)
	}

	if started.Load() {
		t.Error("lazy service OnStart should not run before container Start")
	}

	ctx := context.Background()
	_ = c.Start(ctx)

	if started.Load() {
		t.Error("lazy service OnStart should not run if already instantiated before Start")
	}
}

func TestContainer_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	c := New(WithShutdownTimeout(100 * time.Millisecond))

	var stopped atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "slow"}, nil
		},
		WithOnStop(
			func(ctx context.Context) error {
				select {
				case <-time.After(500 * time.Millisecond):
					stopped.Store(true)
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)

	err := c.Stop(ctx)
	if err == nil {
		t.Error("expected timeout error")
	}

	if stopped.Load() {
		t.Error("slow service should not have completed stop")
	}
}

func TestContainer_ShutdownTimeoutNotSet(t *testing.T) {
	t.Parallel()

	c := New()

	var stopped atomic.Bool

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testService, error) {
			return &testService{name: "test"}, nil
		},
		WithOnStop(
			func(ctx context.Context) error {
				stopped.Store(true)
				return nil
			},
		),
	)

	ctx := context.Background()
	_ = c.Start(ctx)

	err := c.Stop(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !stopped.Load() {
		t.Error("service should have stopped")
	}
}
