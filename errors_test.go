package needle

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/reflect"
)

func TestError_Is_SameCode(t *testing.T) {
	t.Parallel()

	err1 := newError(ErrCodeServiceNotFound, "service A not found", nil)
	err2 := newError(ErrCodeServiceNotFound, "service B not found", nil)

	if !errors.Is(err1, err2) {
		t.Error("errors with same code should match via Is")
	}
}

func TestError_Is_DifferentCode(t *testing.T) {
	t.Parallel()

	err1 := newError(ErrCodeServiceNotFound, "not found", nil)
	err2 := newError(ErrCodeCircularDependency, "cycle", nil)

	if errors.Is(err1, err2) {
		t.Error("errors with different codes should not match via Is")
	}
}

func TestError_Is_DoesNotTraverseTargetChain(t *testing.T) {
	t.Parallel()

	inner := newError(ErrCodeServiceNotFound, "inner", nil)
	wrapper := fmt.Errorf("wrapped: %w", inner)
	check := newError(ErrCodeServiceNotFound, "check", nil)

	if errors.Is(check, wrapper) {
		t.Error("Is should not traverse target's chain, only direct type assertion on target")
	}
}

func TestError_Is_WrappedSource(t *testing.T) {
	t.Parallel()

	inner := newError(ErrCodeServiceNotFound, "inner", nil)
	wrapper := fmt.Errorf("wrapped: %w", inner)
	target := newError(ErrCodeServiceNotFound, "target", nil)

	if !errors.Is(wrapper, target) {
		t.Error("errors.Is should find inner *Error via Unwrap chain of source")
	}
}

// IsNotFound must work against errors that originate inside the internal
// container, even when wrapped through several layers of fmt.Errorf at
// the public/internal seam. Previously this required string-matching the
// error message text -- now the sentinel travels through the chain.
func TestIsNotFound_AcrossInternalSeam(t *testing.T) {
	t.Parallel()

	type unregistered struct{}

	c := New()
	_, err := Invoke[*unregistered](c)
	if err == nil {
		t.Fatal("expected error invoking unregistered service")
	}

	if !IsNotFound(err) {
		t.Errorf("IsNotFound should detect not-found across the internal seam, got: %v", err)
	}

	if !errors.Is(err, container.ErrServiceNotFound) {
		t.Errorf("errors.Is should walk to ErrServiceNotFound, got: %v", err)
	}
}

func TestIsCircularDependency_AcrossInternalSeam(t *testing.T) {
	t.Parallel()

	type circA struct{}
	type circB struct{}

	c := New()
	_ = Register(c, Spec[*circA]{
		Provider: func(ctx context.Context, _ *Container) (*circA, error) {
			return &circA{}, nil
		},
		Dependencies: []string{reflect.TypeKey[*circB]()},
	})
	err := Register(c, Spec[*circB]{
		Provider: func(ctx context.Context, _ *Container) (*circB, error) {
			return &circB{}, nil
		},
		Dependencies: []string{reflect.TypeKey[*circA]()},
	})
	if err == nil {
		t.Fatal("expected circular dependency error")
	}

	if !IsCircularDependency(err) {
		t.Errorf("IsCircularDependency should detect cycle across the internal seam, got: %v", err)
	}
}

func TestIsDuplicateService_AcrossInternalSeam(t *testing.T) {
	t.Parallel()

	type svc struct{}

	c := New()
	_ = Register(c, SpecValue(&svc{}))
	err := Register(c, SpecValue(&svc{}))
	if err == nil {
		t.Fatal("expected duplicate service error")
	}

	if !IsDuplicateService(err) {
		t.Errorf("IsDuplicateService should detect duplicate across the internal seam, got: %v", err)
	}
}
