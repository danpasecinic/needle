package needle

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestScope_Singleton(t *testing.T) {
	t.Parallel()

	c := New()

	var callCount atomic.Int32

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testCounter, error) {
			callCount.Add(1)
			return &testCounter{id: int(callCount.Load())}, nil
		},
	)

	first, _ := Invoke[*testCounter](c)
	second, _ := Invoke[*testCounter](c)
	third, _ := Invoke[*testCounter](c)

	if first != second || second != third {
		t.Error("singleton should return same instance")
	}

	if callCount.Load() != 1 {
		t.Errorf("expected provider to be called once, got %d", callCount.Load())
	}
}

func TestScope_Transient(t *testing.T) {
	t.Parallel()

	c := New()

	var callCount atomic.Int32

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testCounter, error) {
			callCount.Add(1)
			return &testCounter{id: int(callCount.Load())}, nil
		}, WithScope(Transient),
	)

	ctx := context.Background()

	first, _ := InvokeCtx[*testCounter](ctx, c)
	second, _ := InvokeCtx[*testCounter](ctx, c)
	third, _ := InvokeCtx[*testCounter](ctx, c)

	if first == second || second == third {
		t.Error("transient should return different instances")
	}

	if first.id == second.id || second.id == third.id {
		t.Error("transient instances should have different ids")
	}

	if callCount.Load() != 3 {
		t.Errorf("expected provider to be called 3 times, got %d", callCount.Load())
	}
}

func TestScope_Request(t *testing.T) {
	t.Parallel()

	c := New()

	var callCount atomic.Int32

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testCounter, error) {
			callCount.Add(1)
			return &testCounter{id: int(callCount.Load())}, nil
		}, WithScope(Request),
	)

	ctx1 := WithRequestScope(context.Background())
	ctx2 := WithRequestScope(context.Background())

	first1, _ := InvokeCtx[*testCounter](ctx1, c)
	second1, _ := InvokeCtx[*testCounter](ctx1, c)

	first2, _ := InvokeCtx[*testCounter](ctx2, c)
	second2, _ := InvokeCtx[*testCounter](ctx2, c)

	if first1 != second1 {
		t.Error("same request scope should return same instance")
	}

	if first2 != second2 {
		t.Error("same request scope should return same instance")
	}

	if first1 == first2 {
		t.Error("different request scopes should return different instances")
	}

	if callCount.Load() != 2 {
		t.Errorf("expected provider to be called 2 times, got %d", callCount.Load())
	}
}

func TestScope_Request_NoScope(t *testing.T) {
	t.Parallel()

	c := New()

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testCounter, error) {
			return &testCounter{id: 1}, nil
		}, WithScope(Request),
	)

	ctx := context.Background()

	_, err := InvokeCtx[*testCounter](ctx, c)
	if err == nil {
		t.Error("expected error when request scope not in context")
	}
}

func TestScope_Pooled(t *testing.T) {
	t.Parallel()

	c := New()

	var callCount atomic.Int32

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testCounter, error) {
			callCount.Add(1)
			return &testCounter{id: int(callCount.Load())}, nil
		}, WithPoolSize(2),
	)

	ctx := context.Background()

	first, _ := InvokeCtx[*testCounter](ctx, c)
	second, _ := InvokeCtx[*testCounter](ctx, c)

	if callCount.Load() != 2 {
		t.Errorf("expected 2 new instances, got %d", callCount.Load())
	}

	c.Release("*github.com/danpasecinic/needle.testCounter", first)
	c.Release("*github.com/danpasecinic/needle.testCounter", second)

	third, _ := InvokeCtx[*testCounter](ctx, c)
	fourth, _ := InvokeCtx[*testCounter](ctx, c)

	if callCount.Load() != 2 {
		t.Errorf("expected no new instances after release, got %d total calls", callCount.Load())
	}

	if third != first && third != second {
		t.Error("pooled should reuse released instance")
	}

	if fourth != first && fourth != second {
		t.Error("pooled should reuse released instance")
	}
}

func TestScope_Pooled_Overflow(t *testing.T) {
	t.Parallel()

	c := New()

	var callCount atomic.Int32

	_ = Provide(
		c, func(ctx context.Context, r Resolver) (*testCounter, error) {
			callCount.Add(1)
			return &testCounter{id: int(callCount.Load())}, nil
		}, WithPoolSize(1),
	)

	ctx := context.Background()

	first, _ := InvokeCtx[*testCounter](ctx, c)
	second, _ := InvokeCtx[*testCounter](ctx, c)

	c.Release("*github.com/danpasecinic/needle.testCounter", first)
	released := c.Release("*github.com/danpasecinic/needle.testCounter", second)

	if released {
		t.Error("second release should fail (pool full)")
	}
}

type testCounter struct {
	id int
}
