package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/fx"

	"github.com/danpasecinic/needle"
)

func BenchmarkLifecycle_10_Needle(b *testing.B) {
	benchmarkLifecycleNeedle(b, 10, false)
}

func BenchmarkLifecycle_10_NeedleParallel(b *testing.B) {
	benchmarkLifecycleNeedle(b, 10, true)
}

func BenchmarkLifecycle_10_Fx(b *testing.B) {
	benchmarkLifecycleFx(b, 10)
}

func BenchmarkLifecycle_50_Needle(b *testing.B) {
	benchmarkLifecycleNeedle(b, 50, false)
}

func BenchmarkLifecycle_50_NeedleParallel(b *testing.B) {
	benchmarkLifecycleNeedle(b, 50, true)
}

func BenchmarkLifecycle_50_Fx(b *testing.B) {
	benchmarkLifecycleFx(b, 50)
}

func BenchmarkLifecycleWithWork_10_Needle(b *testing.B) {
	benchmarkLifecycleNeedleWithWork(b, 10, false, time.Millisecond)
}

func BenchmarkLifecycleWithWork_10_NeedleParallel(b *testing.B) {
	benchmarkLifecycleNeedleWithWork(b, 10, true, time.Millisecond)
}

func BenchmarkLifecycleWithWork_10_Fx(b *testing.B) {
	benchmarkLifecycleFxWithWork(b, 10, time.Millisecond)
}

func BenchmarkLifecycleWithWork_50_Needle(b *testing.B) {
	benchmarkLifecycleNeedleWithWork(b, 50, false, time.Millisecond)
}

func BenchmarkLifecycleWithWork_50_NeedleParallel(b *testing.B) {
	benchmarkLifecycleNeedleWithWork(b, 50, true, time.Millisecond)
}

func BenchmarkLifecycleWithWork_50_Fx(b *testing.B) {
	benchmarkLifecycleFxWithWork(b, 50, time.Millisecond)
}

func benchmarkLifecycleNeedle(b *testing.B, count int, parallel bool) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var opts []needle.Option
		if parallel {
			opts = append(opts, needle.WithParallel())
		}
		c := needle.New(opts...)

		for j := 0; j < count; j++ {
			idx := j
			key := fmt.Sprintf("svc_%d", j)
			_ = needle.ProvideNamed(
				c, key, func(ctx context.Context, r needle.Resolver) (*Config, error) {
					return &Config{Port: idx}, nil
				},
			)
		}

		ctx := context.Background()
		b.StartTimer()
		_ = c.Start(ctx)
		_ = c.Stop(ctx)
	}
}

func benchmarkLifecycleNeedleWithWork(b *testing.B, count int, parallel bool, work time.Duration) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		opts := []needle.Option{}
		if parallel {
			opts = append(opts, needle.WithParallel())
		}
		c := needle.New(opts...)

		for j := 0; j < count; j++ {
			idx := j
			key := fmt.Sprintf("svc_%d", j)
			_ = needle.ProvideNamed(
				c, key, func(ctx context.Context, r needle.Resolver) (*Config, error) {
					return &Config{Port: idx}, nil
				},
				needle.WithOnStart(
					func(ctx context.Context) error {
						time.Sleep(work)
						return nil
					},
				),
				needle.WithOnStop(
					func(ctx context.Context) error {
						time.Sleep(work)
						return nil
					},
				),
			)
		}

		ctx := context.Background()
		b.StartTimer()
		_ = c.Start(ctx)
		_ = c.Stop(ctx)
	}
}

func benchmarkLifecycleFx(b *testing.B, count int) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		providers := make([]fx.Option, count)
		for j := 0; j < count; j++ {
			idx := j
			name := fmt.Sprintf("svc_%d", j)
			providers[j] = fx.Provide(
				fx.Annotate(
					func() *Config { return &Config{Port: idx} },
					fx.ResultTags(fmt.Sprintf(`name:"%s"`, name)),
				),
			)
		}

		invokers := make([]any, count)
		for j := 0; j < count; j++ {
			name := fmt.Sprintf("svc_%d", j)
			invokers[j] = fx.Annotate(
				func(*Config) {},
				fx.ParamTags(fmt.Sprintf(`name:"%s"`, name)),
			)
		}

		opts := []fx.Option{fx.NopLogger, fx.Invoke(invokers...)}
		opts = append(opts, providers...)
		app := fx.New(opts...)

		ctx := context.Background()
		b.StartTimer()
		_ = app.Start(ctx)
		_ = app.Stop(ctx)
	}
}

func benchmarkLifecycleFxWithWork(b *testing.B, count int, work time.Duration) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		providers := make([]fx.Option, count)
		for j := 0; j < count; j++ {
			idx := j
			name := fmt.Sprintf("svc_%d", j)
			providers[j] = fx.Provide(
				fx.Annotate(
					func(lc fx.Lifecycle) *Config {
						cfg := &Config{Port: idx}
						lc.Append(
							fx.Hook{
								OnStart: func(ctx context.Context) error {
									time.Sleep(work)
									return nil
								},
								OnStop: func(ctx context.Context) error {
									time.Sleep(work)
									return nil
								},
							},
						)
						return cfg
					},
					fx.ResultTags(fmt.Sprintf(`name:"%s"`, name)),
				),
			)
		}

		invokers := make([]any, count)
		for j := 0; j < count; j++ {
			name := fmt.Sprintf("svc_%d", j)
			invokers[j] = fx.Annotate(
				func(*Config) {},
				fx.ParamTags(fmt.Sprintf(`name:"%s"`, name)),
			)
		}

		opts := []fx.Option{fx.NopLogger, fx.Invoke(invokers...)}
		opts = append(opts, providers...)
		app := fx.New(opts...)

		ctx := context.Background()
		b.StartTimer()
		_ = app.Start(ctx)
		_ = app.Stop(ctx)
	}
}
