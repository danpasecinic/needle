package benchmark

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/do/v2"
	"go.uber.org/dig"
	"go.uber.org/fx"

	"github.com/danpasecinic/needle"
)

func BenchmarkNamed_10_Needle(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c := needle.New()
		for j := 0; j < 10; j++ {
			idx := j
			key := fmt.Sprintf("svc_%d", j)
			_ = needle.ProvideNamed(
				c, key, func(ctx context.Context, r needle.Resolver) (*Config, error) {
					return &Config{Port: idx}, nil
				},
			)
		}
	}
}

func BenchmarkNamed_10_Do(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		injector := do.New()
		for j := 0; j < 10; j++ {
			idx := j
			key := fmt.Sprintf("svc_%d", j)
			do.ProvideNamed(
				injector, key, func(i do.Injector) (*Config, error) {
					return &Config{Port: idx}, nil
				},
			)
		}
	}
}

func BenchmarkNamed_10_Dig(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c := dig.New()
		for j := 0; j < 10; j++ {
			idx := j
			name := fmt.Sprintf("svc_%d", j)
			_ = c.Provide(func() *Config { return &Config{Port: idx} }, dig.Name(name))
		}
	}
}

func BenchmarkNamed_10_Fx(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		providers := make([]fx.Option, 10)
		for j := 0; j < 10; j++ {
			idx := j
			name := fmt.Sprintf("svc_%d", j)
			providers[j] = fx.Provide(
				fx.Annotate(
					func() *Config { return &Config{Port: idx} },
					fx.ResultTags(fmt.Sprintf(`name:"%s"`, name)),
				),
			)
		}
		opts := []fx.Option{fx.NopLogger}
		opts = append(opts, providers...)
		_ = fx.New(opts...)
	}
}
