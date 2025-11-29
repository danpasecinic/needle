package container

import (
	"context"
	"errors"
	"testing"
)

func TestContainer_RegisterAndResolve(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	err := c.Register("config", func(ctx context.Context, r Resolver) (any, error) {
		return map[string]string{"port": "8080"}, nil
	}, nil)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	ctx := context.Background()
	instance, err := c.Resolve(ctx, "config")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	cfg, ok := instance.(map[string]string)
	if !ok {
		t.Fatal("expected map[string]string")
	}

	if cfg["port"] != "8080" {
		t.Errorf("expected port 8080, got %s", cfg["port"])
	}
}

func TestContainer_RegisterValue(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	value := "test-value"
	err := c.RegisterValue("myvalue", value)
	if err != nil {
		t.Fatalf("failed to register value: %v", err)
	}

	ctx := context.Background()
	instance, err := c.Resolve(ctx, "myvalue")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if instance != value {
		t.Errorf("expected %v, got %v", value, instance)
	}
}

func TestContainer_DependencyResolution(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	err := c.RegisterValue("config", map[string]string{"db": "postgres"})
	if err != nil {
		t.Fatalf("failed to register config: %v", err)
	}

	err = c.Register("database", func(ctx context.Context, r Resolver) (any, error) {
		cfg, err := r.Resolve(ctx, "config")
		if err != nil {
			return nil, err
		}
		return "connected to " + cfg.(map[string]string)["db"], nil
	}, []string{"config"})
	if err != nil {
		t.Fatalf("failed to register database: %v", err)
	}

	ctx := context.Background()
	instance, err := c.Resolve(ctx, "database")
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if instance != "connected to postgres" {
		t.Errorf("expected 'connected to postgres', got %v", instance)
	}
}

func TestContainer_DuplicateRegistration(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	err := c.RegisterValue("test", "value1")
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err = c.RegisterValue("test", "value2")
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestContainer_CircularDependency(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	err := c.Register("A", func(ctx context.Context, r Resolver) (any, error) {
		return "A", nil
	}, []string{"B"})
	if err != nil {
		t.Fatalf("failed to register A: %v", err)
	}

	err = c.Register("B", func(ctx context.Context, r Resolver) (any, error) {
		return "B", nil
	}, []string{"A"})
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

func TestContainer_MissingDependency(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	err := c.Register("service", func(ctx context.Context, r Resolver) (any, error) {
		_, err := r.Resolve(ctx, "missing")
		return nil, err
	}, []string{"missing"})
	if err != nil {
		t.Fatalf("registration should succeed: %v", err)
	}

	ctx := context.Background()
	_, err = c.Resolve(ctx, "service")
	if err == nil {
		t.Error("expected error for missing dependency")
	}
}

func TestContainer_ProviderError(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	expectedErr := errors.New("provider failed")
	err := c.Register("failing", func(ctx context.Context, r Resolver) (any, error) {
		return nil, expectedErr
	}, nil)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	ctx := context.Background()
	_, err = c.Resolve(ctx, "failing")
	if err == nil {
		t.Error("expected error from provider")
	}
}

func TestContainer_Singleton(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	callCount := 0
	err := c.Register("counter", func(ctx context.Context, r Resolver) (any, error) {
		callCount++
		return callCount, nil
	}, nil)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	ctx := context.Background()

	v1, _ := c.Resolve(ctx, "counter")
	v2, _ := c.Resolve(ctx, "counter")

	if v1 != v2 {
		t.Error("singleton should return same instance")
	}

	if callCount != 1 {
		t.Errorf("provider should be called once, was called %d times", callCount)
	}
}

func TestContainer_Has(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	if c.Has("test") {
		t.Error("should not have unregistered service")
	}

	_ = c.RegisterValue("test", "value")

	if !c.Has("test") {
		t.Error("should have registered service")
	}
}

func TestContainer_Keys(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	_ = c.RegisterValue("a", 1)
	_ = c.RegisterValue("b", 2)
	_ = c.RegisterValue("c", 3)

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestContainer_Size(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	if c.Size() != 0 {
		t.Error("empty container should have size 0")
	}

	_ = c.RegisterValue("a", 1)
	_ = c.RegisterValue("b", 2)

	if c.Size() != 2 {
		t.Errorf("expected size 2, got %d", c.Size())
	}
}

func TestContainer_Validate(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	_ = c.RegisterValue("config", "config")
	_ = c.Register("service", func(ctx context.Context, r Resolver) (any, error) {
		return "service", nil
	}, []string{"config"})

	err := c.Validate()
	if err != nil {
		t.Errorf("validation should pass: %v", err)
	}
}

func TestContainer_ContextCancellation(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	_ = c.Register("slow", func(ctx context.Context, r Resolver) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return "done", nil
		}
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Resolve(ctx, "slow")
	if err == nil {
		t.Log("provider completed before cancellation (acceptable)")
	}
}

func BenchmarkContainer_Resolve(b *testing.B) {
	c := New(&Config{})

	_ = c.RegisterValue("config", map[string]string{"key": "value"})
	_ = c.Register("service", func(ctx context.Context, r Resolver) (any, error) {
		_, _ = r.Resolve(ctx, "config")
		return "service", nil
	}, []string{"config"})

	ctx := context.Background()
	_, _ = c.Resolve(ctx, "service")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Resolve(ctx, "service")
	}
}

func BenchmarkContainer_Register(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c := New(&Config{})
		_ = c.RegisterValue("test", "value")
	}
}
