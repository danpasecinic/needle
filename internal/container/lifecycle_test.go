package container

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestStopService_CollectsAllErrors(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	err1 := errors.New("hook1 failed")
	err2 := errors.New("hook2 failed")

	_ = c.Register("svc", func(ctx context.Context, r Resolver) (any, error) {
		return "instance", nil
	}, nil)

	c.registry.AddOnStop("svc", func(ctx context.Context) error {
		return err1
	})
	c.registry.AddOnStop("svc", func(ctx context.Context) error {
		return err2
	})

	ctx := context.Background()
	_, _ = c.Resolve(ctx, "svc")

	stopErr := c.stopService(ctx, "svc")
	if stopErr == nil {
		t.Fatal("expected error from stopService")
	}

	msg := stopErr.Error()
	if !strings.Contains(msg, "hook1 failed") {
		t.Errorf("expected error to contain 'hook1 failed', got: %s", msg)
	}
	if !strings.Contains(msg, "hook2 failed") {
		t.Errorf("expected error to contain 'hook2 failed', got: %s", msg)
	}
}

func TestStopService_NoErrorWhenHooksSucceed(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	_ = c.Register("svc", func(ctx context.Context, r Resolver) (any, error) {
		return "instance", nil
	}, nil)

	c.registry.AddOnStop("svc", func(ctx context.Context) error {
		return nil
	})

	ctx := context.Background()
	_, _ = c.Resolve(ctx, "svc")

	stopErr := c.stopService(ctx, "svc")
	if stopErr != nil {
		t.Errorf("expected no error, got: %v", stopErr)
	}
}

func TestStartAndStop_Integration(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	var order []string

	_ = c.Register("svc", func(ctx context.Context, r Resolver) (any, error) {
		return "instance", nil
	}, nil)

	c.registry.AddOnStart("svc", func(ctx context.Context) error {
		order = append(order, "started")
		return nil
	})
	c.registry.AddOnStop("svc", func(ctx context.Context) error {
		order = append(order, "stopped")
		return nil
	})

	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if len(order) != 1 || order[0] != "started" {
		t.Errorf("expected [started], got %v", order)
	}

	if err := c.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if len(order) != 2 || order[1] != "stopped" {
		t.Errorf("expected [started, stopped], got %v", order)
	}
}

func TestStopService_MultipleFailingHooks_BothPresent(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	_ = c.Register("svc", func(ctx context.Context, r Resolver) (any, error) {
		return "instance", nil
	}, nil)

	c.registry.AddOnStop("svc", func(ctx context.Context) error {
		return fmt.Errorf("first error")
	})
	c.registry.AddOnStop("svc", func(ctx context.Context) error {
		return fmt.Errorf("second error")
	})

	ctx := context.Background()
	_, _ = c.Resolve(ctx, "svc")

	stopErr := c.stopService(ctx, "svc")
	if stopErr == nil {
		t.Fatal("expected combined error")
	}

	if !strings.Contains(stopErr.Error(), "first error") {
		t.Error("missing first error")
	}
	if !strings.Contains(stopErr.Error(), "second error") {
		t.Error("missing second error")
	}
}
