package needle

import (
	"context"
)

type Hook func(ctx context.Context) error

type Lifecycle struct {
	onStart []Hook
	onStop  []Hook
}

func (l *Lifecycle) Append(other *Lifecycle) {
	if other == nil {
		return
	}
	l.onStart = append(l.onStart, other.onStart...)
	l.onStop = append(l.onStop, other.onStop...)
}

func (l *Lifecycle) OnStart(hook Hook) {
	l.onStart = append(l.onStart, hook)
}

func (l *Lifecycle) OnStop(hook Hook) {
	l.onStop = append(l.onStop, hook)
}

type LifecycleAware interface {
	Lifecycle() *Lifecycle
}
