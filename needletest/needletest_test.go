package needletest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/danpasecinic/needle"
	"github.com/danpasecinic/needle/needletest"
)

type Config struct {
	Port int
	Host string
}

type Database struct {
	Config *Config
}

type UserRepository interface {
	FindByID(id int) string
}

type MockUserRepository struct {
	FindByIDFn func(id int) string
}

func (m *MockUserRepository) FindByID(id int) string {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(id)
	}
	return ""
}

func TestNew(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	if tc == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewWithCleanup(t *testing.T) {
	t.Parallel()

	stopped := make(chan struct{})

	tc := needletest.New(t)
	needletest.MustProvide(tc, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		return &Config{Port: 8080}, nil
	}, needle.WithOnStop(func(ctx context.Context) error {
		close(stopped)
		return nil
	}))

	tc.RequireStart(context.Background())

	select {
	case <-stopped:
		t.Error("stop hook should not be called before test ends")
	default:
	}
}

func TestReplace(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)

	needletest.MustProvideValue(tc, &Config{Port: 8080, Host: "localhost"})
	needletest.MustProvide(tc, func(ctx context.Context, r needle.Resolver) (*Database, error) {
		cfg := needle.MustInvoke[*Config](tc.Container)
		return &Database{Config: cfg}, nil
	})

	needletest.Replace(tc, &Config{Port: 9090, Host: "testhost"})

	db := needletest.MustInvoke[*Database](tc)
	if db.Config.Port != 9090 {
		t.Errorf("expected port 9090, got %d", db.Config.Port)
	}
	if db.Config.Host != "testhost" {
		t.Errorf("expected host testhost, got %s", db.Config.Host)
	}
}

func TestReplaceNamed(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)

	needletest.MustProvideNamedValue(tc, "primary", &Config{Port: 5432})
	needletest.MustProvideNamedValue(tc, "replica", &Config{Port: 5433})

	needletest.ReplaceNamed[*Config](tc, "primary", &Config{Port: 9999})

	primary := needletest.MustInvokeNamed[*Config](tc, "primary")
	if primary.Port != 9999 {
		t.Errorf("expected port 9999, got %d", primary.Port)
	}

	replica := needletest.MustInvokeNamed[*Config](tc, "replica")
	if replica.Port != 5433 {
		t.Errorf("expected port 5433, got %d", replica.Port)
	}
}

func TestReplaceProvider(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)

	needletest.MustProvide(tc, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		return &Config{Port: 8080}, nil
	})

	callCount := 0
	needletest.ReplaceProvider(tc, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		callCount++
		return &Config{Port: 3000}, nil
	})

	cfg := needletest.MustInvoke[*Config](tc)
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
	if callCount != 1 {
		t.Errorf("expected provider to be called once, got %d", callCount)
	}
}

func TestAssertHas(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvideValue(tc, &Config{Port: 8080})

	needletest.AssertHas[*Config](tc)
}

func TestAssertHasNamed(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvideNamedValue(tc, "myconfig", &Config{Port: 8080})

	needletest.AssertHasNamed[*Config](tc, "myconfig")
}

func TestAssertNotHas(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.AssertNotHas[*Config](tc)
}

func TestRequireValidate(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvideValue(tc, &Config{Port: 8080})

	tc.RequireValidate()
}

func TestRequireStartStop(t *testing.T) {
	t.Parallel()

	started := false
	stopped := false

	tc := needletest.New(t)
	needletest.MustProvide(tc, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		return &Config{Port: 8080}, nil
	},
		needle.WithOnStart(func(ctx context.Context) error {
			started = true
			return nil
		}),
		needle.WithOnStop(func(ctx context.Context) error {
			stopped = true
			return nil
		}),
	)

	ctx := context.Background()
	tc.RequireStart(ctx)
	if !started {
		t.Error("expected start hook to be called")
	}

	tc.RequireStop(ctx)
	if !stopped {
		t.Error("expected stop hook to be called")
	}
}

func TestMustInvoke(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvideValue(tc, &Config{Port: 8080, Host: "localhost"})

	cfg := needletest.MustInvoke[*Config](tc)
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", cfg.Host)
	}
}

func TestMustInvokeNamed(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvideNamedValue(tc, "primary", &Config{Port: 5432})

	cfg := needletest.MustInvokeNamed[*Config](tc, "primary")
	if cfg.Port != 5432 {
		t.Errorf("expected port 5432, got %d", cfg.Port)
	}
}

func TestMustProvide(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvide(tc, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		return &Config{Port: 8080}, nil
	})

	needletest.AssertHas[*Config](tc)
}

func TestMustProvideValue(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	needletest.MustProvideValue(tc, &Config{Port: 8080})

	needletest.AssertHas[*Config](tc)
}

func TestMockInjection(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)

	mock := &MockUserRepository{
		FindByIDFn: func(id int) string {
			return "mock-user"
		},
	}

	if err := needle.ProvideValue[UserRepository](tc.Container, mock); err != nil {
		t.Fatalf("failed to provide mock: %v", err)
	}

	repo := needletest.MustInvoke[UserRepository](tc)
	result := repo.FindByID(1)
	if result != "mock-user" {
		t.Errorf("expected 'mock-user', got '%s'", result)
	}
}

func TestReplaceWithMock(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)

	realRepo := &MockUserRepository{
		FindByIDFn: func(id int) string {
			return "real-user"
		},
	}
	if err := needle.ProvideValue[UserRepository](tc.Container, realRepo); err != nil {
		t.Fatalf("failed to provide real repo: %v", err)
	}

	mockRepo := &MockUserRepository{
		FindByIDFn: func(id int) string {
			return "test-user-" + string(rune('0'+id))
		},
	}
	needletest.Replace[UserRepository](tc, mockRepo)

	repo := needletest.MustInvoke[UserRepository](tc)
	result := repo.FindByID(5)
	if result != "test-user-5" {
		t.Errorf("expected 'test-user-5', got '%s'", result)
	}
}

func TestProviderReturningError(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)
	expectedErr := errors.New("initialization failed")

	if err := needle.Provide(tc.Container, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		return nil, expectedErr
	}); err != nil {
		t.Fatalf("failed to provide: %v", err)
	}

	_, err := needle.Invoke[*Config](tc.Container)
	if err == nil {
		t.Error("expected error from provider")
	}
}

func TestDependencyChainWithReplacement(t *testing.T) {
	t.Parallel()

	tc := needletest.New(t)

	needletest.MustProvideValue(tc, &Config{Port: 8080})
	needletest.MustProvide(tc, func(ctx context.Context, r needle.Resolver) (*Database, error) {
		cfg := needle.MustInvoke[*Config](tc.Container)
		return &Database{Config: cfg}, nil
	})

	needletest.Replace(tc, &Config{Port: 3000})

	db := needletest.MustInvoke[*Database](tc)
	if db.Config.Port != 3000 {
		t.Errorf("expected database to use replaced config with port 3000, got %d", db.Config.Port)
	}
}
