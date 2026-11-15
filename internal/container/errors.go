package container

import "errors"

// Sentinel errors emitted from inside the internal container. Each fmt.Errorf
// site that signals one of these conditions wraps the corresponding sentinel
// via %w so the seam carries a typed identity instead of an English string.
//
// The public package's IsNotFound / IsCircularDependency / IsDuplicateService
// predicates use errors.Is against these sentinels.
var (
	ErrServiceNotFound      = errors.New("service not found")
	ErrCircularDependency   = errors.New("circular dependency")
	ErrCircularResolution   = errors.New("circular resolution")
	ErrDuplicateService     = errors.New("service already registered")
	ErrRequestScopeMissing  = errors.New("request scope not in context")
	ErrContainerStateChange = errors.New("invalid container state transition")
)
