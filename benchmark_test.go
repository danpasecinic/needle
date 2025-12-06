package needle

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkStartup_Sequential_10Services(b *testing.B) {
	benchmarkStartup(b, false, 10, 0)
}

func BenchmarkStartup_Parallel_10Services(b *testing.B) {
	benchmarkStartup(b, true, 10, 0)
}

func BenchmarkStartup_Sequential_50Services(b *testing.B) {
	benchmarkStartup(b, false, 50, 0)
}

func BenchmarkStartup_Parallel_50Services(b *testing.B) {
	benchmarkStartup(b, true, 50, 0)
}

func BenchmarkStartup_Sequential_100Services(b *testing.B) {
	benchmarkStartup(b, false, 100, 0)
}

func BenchmarkStartup_Parallel_100Services(b *testing.B) {
	benchmarkStartup(b, true, 100, 0)
}

func BenchmarkStartupWithWork_Sequential_10Services(b *testing.B) {
	benchmarkStartup(b, false, 10, time.Millisecond)
}

func BenchmarkStartupWithWork_Parallel_10Services(b *testing.B) {
	benchmarkStartup(b, true, 10, time.Millisecond)
}

func BenchmarkStartupWithWork_Sequential_50Services(b *testing.B) {
	benchmarkStartup(b, false, 50, time.Millisecond)
}

func BenchmarkStartupWithWork_Parallel_50Services(b *testing.B) {
	benchmarkStartup(b, true, 50, time.Millisecond)
}

func BenchmarkShutdown_Sequential_10Services(b *testing.B) {
	benchmarkShutdown(b, false, 10, 0)
}

func BenchmarkShutdown_Parallel_10Services(b *testing.B) {
	benchmarkShutdown(b, true, 10, 0)
}

func BenchmarkShutdown_Sequential_50Services(b *testing.B) {
	benchmarkShutdown(b, false, 50, 0)
}

func BenchmarkShutdown_Parallel_50Services(b *testing.B) {
	benchmarkShutdown(b, true, 50, 0)
}

func BenchmarkShutdownWithWork_Sequential_10Services(b *testing.B) {
	benchmarkShutdown(b, false, 10, time.Millisecond)
}

func BenchmarkShutdownWithWork_Parallel_10Services(b *testing.B) {
	benchmarkShutdown(b, true, 10, time.Millisecond)
}

func BenchmarkLifecycle_Sequential_Chain5(b *testing.B) {
	benchmarkDependencyChain(b, false, 5, 0)
}

func BenchmarkLifecycle_Parallel_Chain5(b *testing.B) {
	benchmarkDependencyChain(b, true, 5, 0)
}

func BenchmarkLifecycle_Sequential_Chain10(b *testing.B) {
	benchmarkDependencyChain(b, false, 10, 0)
}

func BenchmarkLifecycle_Parallel_Chain10(b *testing.B) {
	benchmarkDependencyChain(b, true, 10, 0)
}

func BenchmarkLifecycleWithWork_Sequential_Chain5(b *testing.B) {
	benchmarkDependencyChain(b, false, 5, time.Millisecond)
}

func BenchmarkLifecycleWithWork_Parallel_Chain5(b *testing.B) {
	benchmarkDependencyChain(b, true, 5, time.Millisecond)
}

func BenchmarkLifecycle_Sequential_Wide10(b *testing.B) {
	benchmarkWideDependencies(b, false, 10, 0)
}

func BenchmarkLifecycle_Parallel_Wide10(b *testing.B) {
	benchmarkWideDependencies(b, true, 10, 0)
}

func BenchmarkLifecycle_Sequential_Wide50(b *testing.B) {
	benchmarkWideDependencies(b, false, 50, 0)
}

func BenchmarkLifecycle_Parallel_Wide50(b *testing.B) {
	benchmarkWideDependencies(b, true, 50, 0)
}

func BenchmarkLifecycleWithWork_Sequential_Wide10(b *testing.B) {
	benchmarkWideDependencies(b, false, 10, time.Millisecond)
}

func BenchmarkLifecycleWithWork_Parallel_Wide10(b *testing.B) {
	benchmarkWideDependencies(b, true, 10, time.Millisecond)
}

type benchService struct {
	id int
}

func benchmarkStartup(b *testing.B, parallel bool, count int, workDuration time.Duration) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var c *Container
		if parallel {
			c = New(WithParallel())
		} else {
			c = New()
		}

		for j := 0; j < count; j++ {
			idx := j
			key := fmt.Sprintf("svc_%d", j)
			_ = ProvideNamed(
				c, key, func(ctx context.Context, r Resolver) (*benchService, error) {
					return &benchService{id: idx}, nil
				},
				WithOnStart(
					func(ctx context.Context) error {
						if workDuration > 0 {
							time.Sleep(workDuration)
						}
						return nil
					},
				),
			)
		}

		ctx := context.Background()
		b.StartTimer()
		_ = c.Start(ctx)
		b.StopTimer()
		_ = c.Stop(ctx)
	}
}

func benchmarkShutdown(b *testing.B, parallel bool, count int, workDuration time.Duration) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var c *Container
		if parallel {
			c = New(WithParallel())
		} else {
			c = New()
		}

		for j := 0; j < count; j++ {
			idx := j
			key := fmt.Sprintf("svc_%d", j)
			_ = ProvideNamed(
				c, key, func(ctx context.Context, r Resolver) (*benchService, error) {
					return &benchService{id: idx}, nil
				},
				WithOnStop(
					func(ctx context.Context) error {
						if workDuration > 0 {
							time.Sleep(workDuration)
						}
						return nil
					},
				),
			)
		}

		ctx := context.Background()
		_ = c.Start(ctx)
		b.StartTimer()
		_ = c.Stop(ctx)
	}
}

type chainService struct {
	level int
}

func benchmarkDependencyChain(b *testing.B, parallel bool, depth int, workDuration time.Duration) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var c *Container
		if parallel {
			c = New(WithParallel())
		} else {
			c = New()
		}

		prevKey := ""
		for j := 0; j < depth; j++ {
			level := j
			key := fmt.Sprintf("chain_%d", j)
			var deps []string
			if prevKey != "" {
				deps = append(deps, prevKey)
			}

			_ = ProvideNamed(
				c, key, func(ctx context.Context, r Resolver) (*chainService, error) {
					return &chainService{level: level}, nil
				},
				WithDependencies(deps...),
				WithOnStart(
					func(ctx context.Context) error {
						if workDuration > 0 {
							time.Sleep(workDuration)
						}
						return nil
					},
				),
				WithOnStop(
					func(ctx context.Context) error {
						if workDuration > 0 {
							time.Sleep(workDuration)
						}
						return nil
					},
				),
			)
			prevKey = key
		}

		ctx := context.Background()
		b.StartTimer()
		_ = c.Start(ctx)
		_ = c.Stop(ctx)
	}
}

type wideService struct {
	id int
}

type aggregatorService struct{}

func benchmarkWideDependencies(b *testing.B, parallel bool, width int, workDuration time.Duration) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var c *Container
		if parallel {
			c = New(WithParallel())
		} else {
			c = New()
		}

		depKeys := make([]string, width)
		for j := 0; j < width; j++ {
			idx := j
			key := fmt.Sprintf("wide_%d", j)
			depKeys[j] = key

			_ = ProvideNamed(
				c, key, func(ctx context.Context, r Resolver) (*wideService, error) {
					return &wideService{id: idx}, nil
				},
				WithOnStart(
					func(ctx context.Context) error {
						if workDuration > 0 {
							time.Sleep(workDuration)
						}
						return nil
					},
				),
				WithOnStop(
					func(ctx context.Context) error {
						if workDuration > 0 {
							time.Sleep(workDuration)
						}
						return nil
					},
				),
			)
		}

		_ = ProvideNamed(
			c, "aggregator", func(ctx context.Context, r Resolver) (*aggregatorService, error) {
				return &aggregatorService{}, nil
			},
			WithDependencies(depKeys...),
			WithOnStart(
				func(ctx context.Context) error {
					if workDuration > 0 {
						time.Sleep(workDuration)
					}
					return nil
				},
			),
			WithOnStop(
				func(ctx context.Context) error {
					if workDuration > 0 {
						time.Sleep(workDuration)
					}
					return nil
				},
			),
		)

		ctx := context.Background()
		b.StartTimer()
		_ = c.Start(ctx)
		_ = c.Stop(ctx)
	}
}
