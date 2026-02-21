package needle

import (
	"errors"
	"fmt"
	"testing"
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
