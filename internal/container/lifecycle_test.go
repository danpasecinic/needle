package container

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStopService_OnStopErrorPropagates(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	stopErr := errors.New("hook failed")

	_ = c.Register(NewServiceEntry(EntryConfig{
		Key: "svc",
		Provider: func(ctx context.Context) (any, error) {
			return "instance", nil
		},
		OnStop: func(ctx context.Context) error {
			return stopErr
		},
	}))

	ctx := context.Background()
	_, _ = c.Resolve(ctx, "svc")

	err := c.stopService(ctx, "svc")
	if err == nil {
		t.Fatal("expected error from stopService")
	}

	if !strings.Contains(err.Error(), "hook failed") {
		t.Errorf("expected error to contain 'hook failed', got: %s", err.Error())
	}
}

func TestStopService_NoErrorWhenHookSucceeds(t *testing.T) {
	t.Parallel()

	c := New(&Config{})

	_ = c.Register(NewServiceEntry(EntryConfig{
		Key: "svc",
		Provider: func(ctx context.Context) (any, error) {
			return "instance", nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	}))

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

	_ = c.Register(NewServiceEntry(EntryConfig{
		Key: "svc",
		Provider: func(ctx context.Context) (any, error) {
			return "instance", nil
		},
		OnStart: func(ctx context.Context) error {
			order = append(order, "started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			order = append(order, "stopped")
			return nil
		},
	}))

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
