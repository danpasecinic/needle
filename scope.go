package needle

import (
	"context"

	"github.com/danpasecinic/needle/internal/container"
	"github.com/danpasecinic/needle/internal/scope"
)

type Scope = scope.Scope

const (
	Singleton = scope.Singleton
	Transient = scope.Transient
	Request   = scope.Request
	Pooled    = scope.Pooled
)

func WithRequestScope(ctx context.Context) context.Context {
	return container.WithRequestScope(ctx)
}

func (c *Container) Release(key string, instance any) {
	c.internal.Release(key, instance)
}
