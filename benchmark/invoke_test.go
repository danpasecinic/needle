package benchmark

import (
	"context"
	"testing"

	"github.com/samber/do/v2"
	"go.uber.org/dig"
	"go.uber.org/fx"

	"github.com/danpasecinic/needle"
)

func BenchmarkInvoke_Singleton_Needle(b *testing.B) {
	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Host: "localhost", Port: 8080})
	ctx := context.Background()
	_ = c.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = needle.Invoke[*Config](c)
	}
	_ = c.Stop(ctx)
}

func BenchmarkInvoke_Singleton_Do(b *testing.B) {
	injector := do.New()
	do.ProvideValue(injector, &Config{Host: "localhost", Port: 8080})
	_ = do.MustInvoke[*Config](injector)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = do.MustInvoke[*Config](injector)
	}
}

func BenchmarkInvoke_Singleton_Dig(b *testing.B) {
	c := dig.New()
	_ = c.Provide(func() *Config { return &Config{Host: "localhost", Port: 8080} })
	_ = c.Invoke(func(*Config) {})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = c.Invoke(func(*Config) {})
	}
}

func BenchmarkInvoke_Singleton_Fx(b *testing.B) {
	var cfg *Config
	app := fx.New(
		fx.NopLogger,
		fx.Provide(func() *Config { return &Config{Host: "localhost", Port: 8080} }),
		fx.Populate(&cfg),
	)
	ctx := context.Background()
	_ = app.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cfg
	}
	_ = app.Stop(ctx)
}

func BenchmarkInvoke_Chain_Needle(b *testing.B) {
	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Host: "localhost", Port: 8080})
	_ = needle.ProvideValue(c, &Logger{Level: "info"})
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Database, error) {
			cfg := needle.MustInvoke[*Config](c)
			log := needle.MustInvoke[*Logger](c)
			return &Database{Config: cfg, Logger: log}, nil
		},
	)
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Cache, error) {
			log := needle.MustInvoke[*Logger](c)
			return &Cache{Logger: log}, nil
		},
	)
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Repository, error) {
			db := needle.MustInvoke[*Database](c)
			cache := needle.MustInvoke[*Cache](c)
			return &Repository{DB: db, Cache: cache}, nil
		},
	)
	_ = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Service, error) {
			repo := needle.MustInvoke[*Repository](c)
			log := needle.MustInvoke[*Logger](c)
			return &Service{Repo: repo, Logger: log}, nil
		},
	)
	ctx := context.Background()
	_ = c.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = needle.Invoke[*Service](c)
	}
	_ = c.Stop(ctx)
}

func BenchmarkInvoke_Chain_Do(b *testing.B) {
	injector := do.New()
	do.ProvideValue(injector, &Config{Host: "localhost", Port: 8080})
	do.ProvideValue(injector, &Logger{Level: "info"})
	do.Provide(
		injector, func(i do.Injector) (*Database, error) {
			cfg := do.MustInvoke[*Config](i)
			log := do.MustInvoke[*Logger](i)
			return &Database{Config: cfg, Logger: log}, nil
		},
	)
	do.Provide(
		injector, func(i do.Injector) (*Cache, error) {
			log := do.MustInvoke[*Logger](i)
			return &Cache{Logger: log}, nil
		},
	)
	do.Provide(
		injector, func(i do.Injector) (*Repository, error) {
			db := do.MustInvoke[*Database](i)
			cache := do.MustInvoke[*Cache](i)
			return &Repository{DB: db, Cache: cache}, nil
		},
	)
	do.Provide(
		injector, func(i do.Injector) (*Service, error) {
			repo := do.MustInvoke[*Repository](i)
			log := do.MustInvoke[*Logger](i)
			return &Service{Repo: repo, Logger: log}, nil
		},
	)
	_ = do.MustInvoke[*Service](injector)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = do.MustInvoke[*Service](injector)
	}
}

func BenchmarkInvoke_Chain_Dig(b *testing.B) {
	c := dig.New()
	_ = c.Provide(func() *Config { return &Config{Host: "localhost", Port: 8080} })
	_ = c.Provide(func() *Logger { return &Logger{Level: "info"} })
	_ = c.Provide(func(cfg *Config, log *Logger) *Database { return &Database{Config: cfg, Logger: log} })
	_ = c.Provide(func(log *Logger) *Cache { return &Cache{Logger: log} })
	_ = c.Provide(func(db *Database, cache *Cache) *Repository { return &Repository{DB: db, Cache: cache} })
	_ = c.Provide(func(repo *Repository, log *Logger) *Service { return &Service{Repo: repo, Logger: log} })
	_ = c.Invoke(func(*Service) {})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = c.Invoke(func(*Service) {})
	}
}

func BenchmarkInvoke_Chain_Fx(b *testing.B) {
	var svc *Service
	app := fx.New(
		fx.NopLogger,
		fx.Provide(func() *Config { return &Config{Host: "localhost", Port: 8080} }),
		fx.Provide(func() *Logger { return &Logger{Level: "info"} }),
		fx.Provide(func(cfg *Config, log *Logger) *Database { return &Database{Config: cfg, Logger: log} }),
		fx.Provide(func(log *Logger) *Cache { return &Cache{Logger: log} }),
		fx.Provide(func(db *Database, cache *Cache) *Repository { return &Repository{DB: db, Cache: cache} }),
		fx.Provide(func(repo *Repository, log *Logger) *Service { return &Service{Repo: repo, Logger: log} }),
		fx.Populate(&svc),
	)
	ctx := context.Background()
	_ = app.Start(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = svc
	}
	_ = app.Stop(ctx)
}
