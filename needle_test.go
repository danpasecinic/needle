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

func TestRegisterAndInvoke(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Register(c, needle.Spec[*Config]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return &Config{Port: 8080, Host: "localhost"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

func TestRegisterValue(t *testing.T) {
	t.Parallel()

	c := needle.New()

	config := &Config{Port: 3000, Host: "0.0.0.0"}
	err := needle.Register(c, needle.SpecValue(config))
	if err != nil {
		t.Fatalf("Register value failed: %v", err)
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

	err := needle.Register(c, needle.SpecValue(&Config{Port: 5432, Host: "db.local"}))
	if err != nil {
		t.Fatalf("Register Config value failed: %v", err)
	}

	err = needle.Register(c, needle.Spec[*Database]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Database, error) {
			cfg := needle.MustInvoke[*Config](c)
			return &Database{Config: cfg, Name: "testdb"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register Database failed: %v", err)
	}

	err = needle.Register(c, needle.Spec[*Server]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Server, error) {
			db := needle.MustInvoke[*Database](c)
			cfg := needle.MustInvoke[*Config](c)
			return &Server{DB: db, Config: cfg}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register Server failed: %v", err)
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

	err := needle.Register(c, needle.Spec[*Database]{
		Name: "primary",
		Provider: func(ctx context.Context, r needle.Resolver) (*Database, error) {
			return &Database{Name: "primary"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register primary failed: %v", err)
	}

	err = needle.Register(c, needle.Spec[*Database]{
		Name: "replica",
		Provider: func(ctx context.Context, r needle.Resolver) (*Database, error) {
			return &Database{Name: "replica"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register replica failed: %v", err)
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

	err := needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

	err := needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

	err := needle.Register(c, needle.SpecValue(&Config{}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

	err := needle.Register(c, needle.SpecValue(&Config{}).WithName("myconfig"))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if !needle.HasNamed[*Config](c, "myconfig") {
		t.Error("HasNamed should return true for existing service")
	}
}

func TestProviderError(t *testing.T) {
	t.Parallel()

	c := needle.New()

	expectedErr := errors.New("provider error")
	err := needle.Register(c, needle.Spec[*Config]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return nil, expectedErr
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, err = needle.Invoke[*Config](c)
	if err == nil {
		t.Error("Invoke should return error from provider")
	}
}

func TestContainerValidate(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Register(c, needle.SpecValue(&Config{}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

	_ = needle.Register(c, needle.SpecValue(&Config{}))
	_ = needle.Register(c, needle.SpecValue(&Database{}))

	if c.Size() != 2 {
		t.Errorf("expected size 2, got %d", c.Size())
	}
}

func TestContainerKeys(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.Register(c, needle.SpecValue(&Config{}))
	_ = needle.Register(c, needle.SpecValue(&Database{}))

	keys := c.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestInvokeWithContext(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Register(c, needle.Spec[*Config]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Config, error) {
			return &Config{Port: 8080}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
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

func BenchmarkRegisterAndInvoke(b *testing.B) {
	c := needle.New()
	_ = needle.Register(c, needle.SpecValue(&Config{Port: 8080}))

	b.ReportAllocs()
	for b.Loop() {
		_, _ = needle.Invoke[*Config](c)
	}
}

func BenchmarkMustInvoke(b *testing.B) {
	c := needle.New()
	_ = needle.Register(c, needle.SpecValue(&Config{Port: 8080}))

	b.ReportAllocs()
	for b.Loop() {
		_ = needle.MustInvoke[*Config](c)
	}
}

func TestOptionalPresent(t *testing.T) {
	t.Parallel()

	c := needle.New()
	_ = needle.Register(c, needle.SpecValue(&Config{Port: 8080, Host: "localhost"}))

	opt, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	opt, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	opt, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defaultCfg := &Config{Port: 3000}

	result := opt.OrElse(defaultCfg)
	if result.Port != 3000 {
		t.Errorf("expected port 3000, got %d", result.Port)
	}

	_ = needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
	opt2, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result2 := opt2.OrElse(defaultCfg)
	if result2.Port != 8080 {
		t.Errorf("expected port 8080, got %d", result2.Port)
	}
}

func TestOptionalOrElseFunc(t *testing.T) {
	t.Parallel()

	c := needle.New()
	callCount := 0

	opt, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	_ = needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
	opt2, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	_ = needle.Register(c, needle.SpecValue(&Config{Port: 5432}).WithName("primary"))

	opt, err := needle.InvokeOptionalNamed[*Config](c, "primary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opt.Present() {
		t.Error("expected primary config to be present")
	}
	if opt.Value().Port != 5432 {
		t.Errorf("expected port 5432, got %d", opt.Value().Port)
	}

	optMissing, err := needle.InvokeOptionalNamed[*Config](c, "replica")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

	_ = needle.Register(c, needle.Spec[*Service]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Service, error) {
			cacheOpt, err := needle.InvokeOptional[*Cache](c)
			if err != nil {
				return nil, err
			}
			return &Service{
				Cache: cacheOpt.OrElse(nil),
			}, nil
		},
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

	_ = needle.Register(c, needle.SpecValue(&Cache{Enabled: true}))
	_ = needle.Register(c, needle.Spec[*Service]{
		Provider: func(ctx context.Context, r needle.Resolver) (*Service, error) {
			cacheOpt, err := needle.InvokeOptional[*Cache](c)
			if err != nil {
				return nil, err
			}
			return &Service{
				Cache: cacheOpt.OrElse(nil),
			}, nil
		},
	})

	svc := needle.MustInvoke[*Service](c)
	if svc.Cache == nil {
		t.Error("expected cache to be present")
	}
	if !svc.Cache.Enabled {
		t.Error("expected cache to be enabled")
	}
}

func TestOptionalResolutionError(t *testing.T) {
	t.Parallel()

	c := needle.New()

	_ = needle.Register(c, needle.Spec[*Config]{
		Provider: func(_ context.Context, _ needle.Resolver) (*Config, error) {
			return nil, errors.New("provider broken")
		},
	})

	opt, err := needle.InvokeOptional[*Config](c)
	if err == nil {
		t.Fatal("expected error for broken provider")
	}
	if opt.Present() {
		t.Error("expected optional to not be present on error")
	}
	if !needle.IsResolutionFailed(err) {
		t.Errorf("expected resolution failed error, got: %v", err)
	}
}

func TestOptionalNotRegisteredNoError(t *testing.T) {
	t.Parallel()

	c := needle.New()

	opt, err := needle.InvokeOptional[*Config](c)
	if err != nil {
		t.Fatalf("expected no error for unregistered service, got: %v", err)
	}
	if opt.Present() {
		t.Error("expected optional to not be present")
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
