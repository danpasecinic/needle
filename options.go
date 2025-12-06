package needle

import (
	"log/slog"
	"time"
)

type Option func(*containerConfig)

func WithLogger(logger *slog.Logger) Option {
	return func(cfg *containerConfig) {
		cfg.logger = logger
	}
}

func WithResolveObserver(hook ResolveHook) Option {
	return func(cfg *containerConfig) {
		cfg.onResolve = append(cfg.onResolve, hook)
	}
}

func WithProvideObserver(hook ProvideHook) Option {
	return func(cfg *containerConfig) {
		cfg.onProvide = append(cfg.onProvide, hook)
	}
}

func WithStartObserver(hook StartHook) Option {
	return func(cfg *containerConfig) {
		cfg.onStart = append(cfg.onStart, hook)
	}
}

func WithStopObserver(hook StopHook) Option {
	return func(cfg *containerConfig) {
		cfg.onStop = append(cfg.onStop, hook)
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(cfg *containerConfig) {
		cfg.shutdownTimeout = timeout
	}
}

func WithParallel() Option {
	return func(cfg *containerConfig) {
		cfg.parallel = true
	}
}
