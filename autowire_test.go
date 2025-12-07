package needle_test

import (
	"context"
	"errors"
	"testing"

	"github.com/danpasecinic/needle"
)

type TestLogger struct {
	Name string
}

type TestDatabase struct {
	URL string
}

type TestCache struct {
	Size int
}

type TestServiceWithTags struct {
	Logger *TestLogger   `needle:""`
	DB     *TestDatabase `needle:""`
	Cache  *TestCache    `needle:",optional"`
}

type TestServiceWithNamedDep struct {
	Primary   *TestDatabase `needle:"primary"`
	Secondary *TestDatabase `needle:"secondary,optional"`
}

func TestInvokeStruct(t *testing.T) {
	t.Run(
		"resolves tagged fields", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "app"})
			needle.ProvideValue(c, &TestDatabase{URL: "postgres://localhost"})

			svc, err := needle.InvokeStruct[*TestServiceWithTags](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc.Logger == nil || svc.Logger.Name != "app" {
				t.Error("Logger not injected correctly")
			}
			if svc.DB == nil || svc.DB.URL != "postgres://localhost" {
				t.Error("DB not injected correctly")
			}
			if svc.Cache != nil {
				t.Error("Cache should be nil (optional, not provided)")
			}
		},
	)

	t.Run(
		"resolves named dependencies", func(t *testing.T) {
			c := needle.New()

			needle.ProvideNamedValue(c, "primary", &TestDatabase{URL: "primary-db"})
			needle.ProvideNamedValue(c, "secondary", &TestDatabase{URL: "secondary-db"})

			svc, err := needle.InvokeStruct[*TestServiceWithNamedDep](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc.Primary == nil || svc.Primary.URL != "primary-db" {
				t.Error("Primary DB not injected correctly")
			}
			if svc.Secondary == nil || svc.Secondary.URL != "secondary-db" {
				t.Error("Secondary DB not injected correctly")
			}
		},
	)

	t.Run(
		"fails on missing required dependency", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "app"})

			_, err := needle.InvokeStruct[*TestServiceWithTags](c)
			if err == nil {
				t.Fatal("expected error for missing required dependency")
			}
		},
	)

	t.Run(
		"succeeds with missing optional dependency", func(t *testing.T) {
			c := needle.New()

			needle.ProvideNamedValue(c, "primary", &TestDatabase{URL: "primary-db"})

			svc, err := needle.InvokeStruct[*TestServiceWithNamedDep](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc.Primary == nil {
				t.Error("Primary should be set")
			}
			if svc.Secondary != nil {
				t.Error("Secondary should be nil (optional, not provided)")
			}
		},
	)

	t.Run(
		"returns non-pointer struct", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "app"})
			needle.ProvideValue(c, &TestDatabase{URL: "postgres://localhost"})

			svc, err := needle.InvokeStruct[TestServiceWithTags](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc.Logger == nil || svc.Logger.Name != "app" {
				t.Error("Logger not injected correctly")
			}
		},
	)
}

func NewTestLogger() *TestLogger {
	return &TestLogger{Name: "default"}
}

func NewTestDatabase(logger *TestLogger) *TestDatabase {
	return &TestDatabase{URL: "db-for-" + logger.Name}
}

func NewTestDatabaseWithError(logger *TestLogger) (*TestDatabase, error) {
	if logger.Name == "fail" {
		return nil, errors.New("intentional failure")
	}
	return &TestDatabase{URL: "db-for-" + logger.Name}, nil
}

type TestUserService struct {
	DB     *TestDatabase
	Logger *TestLogger
}

func NewTestUserService(db *TestDatabase, logger *TestLogger) *TestUserService {
	return &TestUserService{DB: db, Logger: logger}
}

func TestProvideFunc(t *testing.T) {
	t.Run(
		"auto-wires constructor parameters", func(t *testing.T) {
			c := needle.New()

			needle.ProvideFunc[*TestLogger](c, NewTestLogger)
			needle.ProvideFunc[*TestDatabase](c, NewTestDatabase)

			db, err := needle.Invoke[*TestDatabase](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if db.URL != "db-for-default" {
				t.Errorf("expected URL 'db-for-default', got '%s'", db.URL)
			}
		},
	)

	t.Run(
		"handles constructor returning error", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "fail"})
			needle.ProvideFunc[*TestDatabase](c, NewTestDatabaseWithError)

			_, err := needle.Invoke[*TestDatabase](c)
			if err == nil {
				t.Fatal("expected error from constructor")
			}
		},
	)

	t.Run(
		"chains multiple auto-wired services", func(t *testing.T) {
			c := needle.New()

			needle.ProvideFunc[*TestLogger](c, NewTestLogger)
			needle.ProvideFunc[*TestDatabase](c, NewTestDatabase)
			needle.ProvideFunc[*TestUserService](c, NewTestUserService)

			svc, err := needle.Invoke[*TestUserService](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc.Logger == nil || svc.Logger.Name != "default" {
				t.Error("Logger not injected correctly")
			}
			if svc.DB == nil || svc.DB.URL != "db-for-default" {
				t.Error("DB not injected correctly")
			}
		},
	)

	t.Run(
		"fails on missing dependency", func(t *testing.T) {
			c := needle.New()

			needle.ProvideFunc[*TestDatabase](c, NewTestDatabase)

			_, err := needle.Invoke[*TestDatabase](c)
			if err == nil {
				t.Fatal("expected error for missing Logger dependency")
			}
		},
	)

	t.Run(
		"works with zero-arg constructor", func(t *testing.T) {
			c := needle.New()

			needle.ProvideFunc[*TestLogger](c, NewTestLogger)

			logger, err := needle.Invoke[*TestLogger](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if logger.Name != "default" {
				t.Errorf("expected Name 'default', got '%s'", logger.Name)
			}
		},
	)
}

func TestProvideStruct(t *testing.T) {
	t.Run(
		"registers struct with tagged fields", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "app"})
			needle.ProvideValue(c, &TestDatabase{URL: "postgres://localhost"})
			needle.ProvideStruct[*TestServiceWithTags](c)

			svc, err := needle.Invoke[*TestServiceWithTags](c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if svc.Logger == nil || svc.Logger.Name != "app" {
				t.Error("Logger not injected correctly")
			}
		},
	)

	t.Run(
		"validates dependencies on registration", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "app"})
			needle.ProvideStruct[*TestServiceWithTags](c)

			err := c.Validate()
			if err == nil {
				t.Fatal("expected validation error for missing DB dependency")
			}
		},
	)
}

func TestProvideStructWithContext(t *testing.T) {
	c := needle.New()

	needle.ProvideValue(c, &TestLogger{Name: "ctx-test"})
	needle.ProvideValue(c, &TestDatabase{URL: "ctx-db"})

	ctx := context.Background()
	svc, err := needle.InvokeStructCtx[*TestServiceWithTags](ctx, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.Logger.Name != "ctx-test" {
		t.Error("context-based invocation failed")
	}
}

func TestMustProvideFunc(t *testing.T) {
	t.Run(
		"panics on invalid constructor", func(t *testing.T) {
			c := needle.New()

			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic")
				}
			}()

			needle.MustProvideFunc[*TestLogger](c, "not a function")
		},
	)
}

func TestMustProvideStruct(t *testing.T) {
	t.Run(
		"does not panic on valid struct", func(t *testing.T) {
			c := needle.New()

			needle.ProvideValue(c, &TestLogger{Name: "app"})
			needle.ProvideValue(c, &TestDatabase{URL: "db"})

			needle.MustProvideStruct[*TestServiceWithTags](c)

			if !needle.Has[*TestServiceWithTags](c) {
				t.Error("struct should be registered")
			}
		},
	)
}
