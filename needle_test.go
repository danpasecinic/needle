package needle_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/danpasecinic/needle"
)

type Config struct {
	Port int
	Host string
}

type Database struct {
	Config *Config
	Name   string
}

type Server struct {
	DB     *Database
	Config *Config
}

func TestNew(t *testing.T) {
	t.Parallel()

	c := needle.New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewWithLogger(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	c := needle.New(needle.WithLogger(logger))
	if c == nil {
		t.Fatal("New() with logger returned nil")
	}
}

func TestProvideAndInvoke(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return &Config{Port: 8080, Host: "localhost"}, nil
		},
	)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	cfg, err := needle.Invoke[*Config](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", cfg.Host)
	}
}

func TestProvideValue(t *testing.T) {
	t.Parallel()

	c := needle.New()

	config := &Config{Port: 3000, Host: "0.0.0.0"}
	err := needle.ProvideValue(c, config)
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	cfg, err := needle.Invoke[*Config](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if cfg != config {
		t.Error("expected same instance")
	}
}

func TestDependencyChain(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideValue(c, &Config{Port: 5432, Host: "db.local"})
	if err != nil {
		t.Fatalf("ProvideValue for Config failed: %v", err)
	}

	err = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Database, error) {
			cfg := needle.MustInvoke[*Config](c)
			return &Database{Config: cfg, Name: "testdb"}, nil
		},
	)
	if err != nil {
		t.Fatalf("Provide for Database failed: %v", err)
	}

	err = needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
			db := needle.MustInvoke[*Database](c)
			cfg := needle.MustInvoke[*Config](c)
			return &Server{DB: db, Config: cfg}, nil
		},
	)
	if err != nil {
		t.Fatalf("Provide for Server failed: %v", err)
	}

	server, err := needle.Invoke[*Server](c)
	if err != nil {
		t.Fatalf("Invoke for Server failed: %v", err)
	}

	if server.DB == nil {
		t.Error("server.DB should not be nil")
	}
	if server.Config == nil {
		t.Error("server.Config should not be nil")
	}
	if server.DB.Config != server.Config {
		t.Error("Database and Server should share the same Config")
	}
}

func TestNamedServices(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideNamed(
		c, "primary", func(ctx context.Context, r needle.Resolver) (*Database, error) {
			return &Database{Name: "primary"}, nil
		},
	)
	if err != nil {
		t.Fatalf("ProvideNamed for primary failed: %v", err)
	}

	err = needle.ProvideNamed(
		c, "replica", func(ctx context.Context, r needle.Resolver) (*Database, error) {
			return &Database{Name: "replica"}, nil
		},
	)
	if err != nil {
		t.Fatalf("ProvideNamed for replica failed: %v", err)
	}

	primary, err := needle.InvokeNamed[*Database](c, "primary")
	if err != nil {
		t.Fatalf("InvokeNamed for primary failed: %v", err)
	}

	replica, err := needle.InvokeNamed[*Database](c, "replica")
	if err != nil {
		t.Fatalf("InvokeNamed for replica failed: %v", err)
	}

	if primary.Name != "primary" {
		t.Errorf("expected 'primary', got %s", primary.Name)
	}
	if replica.Name != "replica" {
		t.Errorf("expected 'replica', got %s", replica.Name)
	}
}

func TestMustInvoke(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideValue(c, &Config{Port: 8080})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	cfg := needle.MustInvoke[*Config](c)
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
}

func TestMustInvokePanics(t *testing.T) {
	t.Parallel()

	c := needle.New()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustInvoke should panic for missing service")
		}
	}()

	needle.MustInvoke[*Config](c)
}

func TestTryInvoke(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_, ok := needle.TryInvoke[*Config](c)
	if ok {
		t.Error("TryInvoke should return false for missing service")
	}

	err := needle.ProvideValue(c, &Config{Port: 8080})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	cfg, ok := needle.TryInvoke[*Config](c)
	if !ok {
		t.Error("TryInvoke should return true for existing service")
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
}

func TestHas(t *testing.T) {
	t.Parallel()

	c := needle.New()

	if needle.Has[*Config](c) {
		t.Error("Has should return false for missing service")
	}

	err := needle.ProvideValue(c, &Config{})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	if !needle.Has[*Config](c) {
		t.Error("Has should return true for existing service")
	}
}

func TestHasNamed(t *testing.T) {
	t.Parallel()

	c := needle.New()

	if needle.HasNamed[*Config](c, "myconfig") {
		t.Error("HasNamed should return false for missing service")
	}

	err := needle.ProvideNamedValue(c, "myconfig", &Config{})
	if err != nil {
		t.Fatalf("ProvideNamedValue failed: %v", err)
	}

	if !needle.HasNamed[*Config](c, "myconfig") {
		t.Error("HasNamed should return true for existing service")
	}
}

func TestProviderError(t *testing.T) {
	t.Parallel()

	c := needle.New()

	expectedErr := errors.New("provider error")
	err := needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return nil, expectedErr
		},
	)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	_, err = needle.Invoke[*Config](c)
	if err == nil {
		t.Error("Invoke should return error from provider")
	}
}

func TestContainerValidate(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideValue(c, &Config{})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	err = c.Validate()
	if err != nil {
		t.Errorf("Validate should pass: %v", err)
	}
}

func TestContainerSize(t *testing.T) {
	t.Parallel()

	c := needle.New()

	if c.Size() != 0 {
		t.Error("empty container should have size 0")
	}

	_ = needle.ProvideValue(c, &Config{})
	_ = needle.ProvideValue(c, &Database{})

	if c.Size() != 2 {
		t.Errorf("expected size 2, got %d", c.Size())
	}
}

func TestContainerKeys(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.ProvideValue(c, &Config{})
	_ = needle.ProvideValue(c, &Database{})

	keys := c.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestInvokeWithContext(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Provide(
		c, func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return &Config{Port: 8080}, nil
		},
	)
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	ctx := context.Background()
	cfg, err := needle.InvokeCtx[*Config](ctx, c)
	if err != nil {
		t.Fatalf("InvokeCtx failed: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
}

func BenchmarkProvideAndInvoke(b *testing.B) {
	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Port: 8080})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = needle.Invoke[*Config](c)
	}
}

func BenchmarkMustInvoke(b *testing.B) {
	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Port: 8080})

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = needle.MustInvoke[*Config](c)
	}
}

func TestOptionalPresent(t *testing.T) {
	t.Parallel()

	c := needle.New()
	_ = needle.ProvideValue(c, &Config{Port: 8080, Host: "localhost"})

	opt := needle.InvokeOptional[*Config](c)

	if !opt.Present() {
		t.Error("expected optional to be present")
	}

	cfg, ok := opt.Get()
	if !ok {
		t.Error("expected Get() to return true")
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}

	if opt.Value().Host != "localhost" {
		t.Errorf("expected host localhost, got %s", opt.Value().Host)
	}
}

func TestOptionalNotPresent(t *testing.T) {
	t.Parallel()

	c := needle.New()

	opt := needle.InvokeOptional[*Config](c)

	if opt.Present() {
		t.Error("expected optional to not be present")
	}

	cfg, ok := opt.Get()
	if ok {
		t.Error("expected Get() to return false")
	}
	if cfg != nil {
		t.Error("expected nil value")
	}
}

func TestOptionalOrElse(t *testing.T) {
	t.Parallel()

	c := needle.New()

	opt := needle.InvokeOptional[*Config](c)
	defaultCfg := &Config{Port: 3000}

	result := opt.OrElse(defaultCfg)
	if result.Port != 3000 {
		t.Errorf("expected port 3000, got %d", result.Port)
	}

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	opt2 := needle.InvokeOptional[*Config](c)

	result2 := opt2.OrElse(defaultCfg)
	if result2.Port != 8080 {
		t.Errorf("expected port 8080, got %d", result2.Port)
	}
}

func TestOptionalOrElseFunc(t *testing.T) {
	t.Parallel()

	c := needle.New()
	callCount := 0

	opt := needle.InvokeOptional[*Config](c)
	result := opt.OrElseFunc(func() *Config {
		callCount++
		return &Config{Port: 9000}
	})

	if result.Port != 9000 {
		t.Errorf("expected port 9000, got %d", result.Port)
	}
	if callCount != 1 {
		t.Errorf("expected func to be called once, got %d", callCount)
	}

	_ = needle.ProvideValue(c, &Config{Port: 8080})
	opt2 := needle.InvokeOptional[*Config](c)
	result2 := opt2.OrElseFunc(func() *Config {
		callCount++
		return &Config{Port: 9000}
	})

	if result2.Port != 8080 {
		t.Errorf("expected port 8080, got %d", result2.Port)
	}
	if callCount != 1 {
		t.Errorf("expected func to not be called again, got %d", callCount)
	}
}

func TestOptionalNamed(t *testing.T) {
	t.Parallel()

	c := needle.New()
	_ = needle.ProvideNamedValue(c, "primary", &Config{Port: 5432})

	opt := needle.InvokeOptionalNamed[*Config](c, "primary")
	if !opt.Present() {
		t.Error("expected primary config to be present")
	}
	if opt.Value().Port != 5432 {
		t.Errorf("expected port 5432, got %d", opt.Value().Port)
	}

	optMissing := needle.InvokeOptionalNamed[*Config](c, "replica")
	if optMissing.Present() {
		t.Error("expected replica config to not be present")
	}
}

func TestOptionalInProvider(t *testing.T) {
	t.Parallel()

	c := needle.New()

	type Cache struct {
		Enabled bool
	}

	type Service struct {
		Cache *Cache
	}

	_ = needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Service, error) {
		cacheOpt := needle.InvokeOptional[*Cache](c)
		return &Service{
			Cache: cacheOpt.OrElse(nil),
		}, nil
	})

	svc := needle.MustInvoke[*Service](c)
	if svc.Cache != nil {
		t.Error("expected cache to be nil when not provided")
	}
}

func TestOptionalInProviderWithValue(t *testing.T) {
	t.Parallel()

	c := needle.New()

	type Cache struct {
		Enabled bool
	}

	type Service struct {
		Cache *Cache
	}

	_ = needle.ProvideValue(c, &Cache{Enabled: true})
	_ = needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Service, error) {
		cacheOpt := needle.InvokeOptional[*Cache](c)
		return &Service{
			Cache: cacheOpt.OrElse(nil),
		}, nil
	})

	svc := needle.MustInvoke[*Service](c)
	if svc.Cache == nil {
		t.Error("expected cache to be present")
	}
	if !svc.Cache.Enabled {
		t.Error("expected cache to be enabled")
	}
}

func TestSomeNone(t *testing.T) {
	t.Parallel()

	some := needle.Some(&Config{Port: 8080})
	if !some.Present() {
		t.Error("Some should be present")
	}
	if some.Value().Port != 8080 {
		t.Errorf("expected port 8080, got %d", some.Value().Port)
	}

	none := needle.None[*Config]()
	if none.Present() {
		t.Error("None should not be present")
	}
}
