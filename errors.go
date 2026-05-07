package needle

import (
	"errors"
	"fmt"
	"strings"

	"github.com/danpasecinic/needle/internal/container"
)

type ErrorCode uint16

const (
	ErrCodeUnknown ErrorCode = iota
	ErrCodeServiceNotFound
	ErrCodeCircularDependency
	ErrCodeDuplicateService
	ErrCodeResolutionFailed
	ErrCodeProviderFailed
	ErrCodeStartupFailed
	ErrCodeShutdownFailed
	ErrCodeHealthCheckFailed
	ErrCodeScopeNotFound
	ErrCodeValidationFailed
	ErrCodeTimeout
	ErrCodeContainerNotStarted
	ErrCodeContainerAlreadyStarted
	ErrCodeModuleApplyFailed
	ErrCodeModuleInvalidProvider
	ErrCodeDecoratorFailed
)

var codeNames = map[ErrorCode]string{
	ErrCodeUnknown:                 "UNKNOWN",
	ErrCodeServiceNotFound:         "SERVICE_NOT_FOUND",
	ErrCodeCircularDependency:      "CIRCULAR_DEPENDENCY",
	ErrCodeDuplicateService:        "DUPLICATE_SERVICE",
	ErrCodeResolutionFailed:        "RESOLUTION_FAILED",
	ErrCodeProviderFailed:          "PROVIDER_FAILED",
	ErrCodeStartupFailed:           "STARTUP_FAILED",
	ErrCodeShutdownFailed:          "SHUTDOWN_FAILED",
	ErrCodeHealthCheckFailed:       "HEALTH_CHECK_FAILED",
	ErrCodeScopeNotFound:           "SCOPE_NOT_FOUND",
	ErrCodeValidationFailed:        "VALIDATION_FAILED",
	ErrCodeTimeout:                 "TIMEOUT",
	ErrCodeContainerNotStarted:     "CONTAINER_NOT_STARTED",
	ErrCodeContainerAlreadyStarted: "CONTAINER_ALREADY_STARTED",
	ErrCodeModuleApplyFailed:       "MODULE_APPLY_FAILED",
	ErrCodeModuleInvalidProvider:   "MODULE_INVALID_PROVIDER",
	ErrCodeDecoratorFailed:         "DECORATOR_FAILED",
}

func (c ErrorCode) String() string {
	if name, ok := codeNames[c]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", c)
}

type Error struct {
	Code    ErrorCode
	Message string
	Service string
	Cause   error
	Stack   []string
}

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s]", e.Code)

	if e.Service != "" {
		fmt.Fprintf(&b, " service=%q:", e.Service)
	}

	b.WriteString(" ")
	b.WriteString(e.Message)

	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}

	return b.String()
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

func (e *Error) WithService(service string) *Error {
	e.Service = service
	return e
}

func (e *Error) WithStack(stack []string) *Error {
	e.Stack = stack
	return e
}

func newError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func errServiceNotFound(serviceType string) *Error {
	return newError(
		ErrCodeServiceNotFound,
		fmt.Sprintf("no provider registered for type %s", serviceType),
		container.ErrServiceNotFound,
	).WithService(serviceType)
}

func errResolutionFailed(serviceType string, cause error) *Error {
	return newError(
		ErrCodeResolutionFailed,
		fmt.Sprintf("failed to resolve %s", serviceType),
		cause,
	).WithService(serviceType)
}

func errStartupFailed(serviceType string, cause error) *Error {
	return newError(
		ErrCodeStartupFailed,
		fmt.Sprintf("failed to start %s", serviceType),
		cause,
	).WithService(serviceType)
}

func errShutdownFailed(serviceType string, cause error) *Error {
	return newError(
		ErrCodeShutdownFailed,
		fmt.Sprintf("failed to stop %s", serviceType),
		cause,
	).WithService(serviceType)
}

func errHealthCheckFailed(serviceType string, cause error) *Error {
	return newError(
		ErrCodeHealthCheckFailed,
		fmt.Sprintf("health check failed for %s", serviceType),
		cause,
	).WithService(serviceType)
}

func IsNotFound(err error) bool {
	return errors.Is(err, container.ErrServiceNotFound)
}

func IsCircularDependency(err error) bool {
	return errors.Is(err, container.ErrCircularDependency) ||
		errors.Is(err, container.ErrCircularResolution)
}

func IsDuplicateService(err error) bool {
	return errors.Is(err, container.ErrDuplicateService)
}

func IsResolutionFailed(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == ErrCodeResolutionFailed
}

func IsProviderFailed(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == ErrCodeProviderFailed
}

func IsStartupFailed(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == ErrCodeStartupFailed
}

func IsShutdownFailed(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == ErrCodeShutdownFailed
}

func IsHealthCheckFailed(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == ErrCodeHealthCheckFailed
}
