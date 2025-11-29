package needle

import "log/slog"

type Option func(*containerConfig)

func WithLogger(logger *slog.Logger) Option {
	return func(cfg *containerConfig) {
		cfg.logger = logger
	}
}
