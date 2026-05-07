package needle

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/danpasecinic/needle/internal/reflect"
)

func TestConcurrentSingletonResolve(t *testing.T) {
	t.Parallel()

	c := New()
	_ = Register(c, SpecValue(&testCounter{id: 42}))

	const n = 100
	results := make([]*testCounter, n)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			val, err := Invoke[*testCounter](c)
			if err != nil {
				t.Errorf("goroutine %d: %v", idx, err)
				return
			}
			results[idx] = val
		}(i)
	}
	wg.Wait()

	for i := 1; i < n; i++ {
		if results[i] != results[0] {
			t.Fatal("singleton must return same instance across goroutines")
		}
	}
}

func TestConcurrentNamedRegisterAndInvoke(t *testing.T) {
	t.Parallel()

	c := New()
	const n = 50

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			_ = Register(c, SpecValue(&concService{id: idx}).WithName(fmt.Sprintf("s%d", idx)))
		}(i)
	}
	wg.Wait()

	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			val, err := InvokeNamed[*concService](c, fmt.Sprintf("s%d", idx))
			if err != nil {
				t.Errorf("invoke s%d: %v", idx, err)
				return
			}
			if val.id != idx {
				t.Errorf("s%d: expected id %d, got %d", idx, idx, val.id)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentPoolAcquireRelease(t *testing.T) {
	t.Parallel()

	c := New()
	var created atomic.Int32

	_ = Register(c, Spec[*testCounter]{
		Provider: func(_ context.Context, _ *Container) (*testCounter, error) {
			return &testCounter{id: int(created.Add(1))}, nil
		},
		Scope:    Pooled,
		PoolSize: 3,
	})

	key := reflect.TypeKey[*testCounter]()

	instances := make([]*testCounter, 3)
	for i := range 3 {
		inst, err := Invoke[*testCounter](c)
		if err != nil {
			t.Fatalf("pre-fill %d: %v", i, err)
		}
		instances[i] = inst
	}
	for _, inst := range instances {
		c.Release(key, inst)
	}

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			inst, err := Invoke[*testCounter](c)
			if err != nil {
				return
			}
			c.Release(key, inst)
		}()
	}
	wg.Wait()

	if created.Load() < 3 {
		t.Errorf("expected at least 3 provider calls, got %d", created.Load())
	}
}

func TestConcurrentTransientDifferentKeys(t *testing.T) {
	t.Parallel()

	c := New()
	const n = 50

	for i := range n {
		idx := i
		_ = Register(c, Spec[*concService]{
			Name: fmt.Sprintf("t%d", idx),
			Provider: func(_ context.Context, _ *Container) (*concService, error) {
				return &concService{id: idx}, nil
			},
			Scope: Transient,
		})
	}

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			val, err := InvokeNamed[*concService](c, fmt.Sprintf("t%d", idx))
			if err != nil {
				t.Errorf("t%d: %v", idx, err)
				return
			}
			if val == nil {
				t.Errorf("t%d: got nil", idx)
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrentRequestScopeIsolation(t *testing.T) {
	t.Parallel()

	c := New()
	var created atomic.Int32

	_ = Register(c, Spec[*testCounter]{
		Provider: func(_ context.Context, _ *Container) (*testCounter, error) {
			return &testCounter{id: int(created.Add(1))}, nil
		},
		Scope: Request,
	})

	const numContexts = 10
	const resolvesPerCtx = 5

	distinct := make(map[*testCounter]bool)

	for ci := range numContexts {
		ctx := WithRequestScope(context.Background())
		first, err := InvokeCtx[*testCounter](ctx, c)
		if err != nil {
			t.Fatalf("ctx %d: %v", ci, err)
		}
		for ri := 1; ri < resolvesPerCtx; ri++ {
			val, err := InvokeCtx[*testCounter](ctx, c)
			if err != nil {
				t.Fatalf("ctx %d resolve %d: %v", ci, ri, err)
			}
			if val != first {
				t.Errorf("ctx %d: resolve %d returned different instance", ci, ri)
			}
		}
		distinct[first] = true
	}

	if len(distinct) != numContexts {
		t.Errorf("expected %d distinct instances, got %d", numContexts, len(distinct))
	}
}

func TestConcurrentReplaceNoRace(t *testing.T) {
	t.Parallel()

	c := New()
	_ = Register(c, SpecValue(&testCounter{id: 0}))

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				_ = Replace(c, SpecValue(&testCounter{id: idx}))
			} else {
				_, _ = Invoke[*testCounter](c)
			}
		}(i)
	}
	wg.Wait()
}

type concService struct {
	id int
}
