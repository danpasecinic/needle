package needle

import "log/slog"

type Option func(*containerConfig)

func WithLogger(logger *slog.Logger) Option {
	return func(cfg *containerConfig) {
		cfg.logger = logger
	}
}

func WithOnResolve(hook ResolveHook) Option {
	return func(cfg *containerConfig) {
		cfg.onResolve = append(cfg.onResolve, hook)
	}
}

func WithOnProvide(hook ProvideHook) Option {
	return func(cfg *containerConfig) {
		cfg.onProvide = append(cfg.onProvide, hook)
	}
}

func WithOnStart(hook StartHook) Option {
	return func(cfg *containerConfig) {
		cfg.onStart = append(cfg.onStart, hook)
	}
}

func WithOnStop(hook StopHook) Option {
	return func(cfg *containerConfig) {
		cfg.onStop = append(cfg.onStop, hook)
	}
}
