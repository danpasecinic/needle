package needle_test

import (
	"context"
	"testing"

	"github.com/danpasecinic/needle"
)

type Logger struct {
	Prefix string
}

type UserRepository interface {
	FindByID(id int) string
}

type PostgresUserRepo struct {
	DB *Database
}

func (r *PostgresUserRepo) FindByID(id int) string {
	return "user-" + r.DB.Name
}

func TestModuleBasic(t *testing.T) {
	t.Parallel()

	module := needle.NewModule("test")
	if module.Name() != "test" {
		t.Errorf("expected module name 'test', got %s", module.Name())
	}
}

func TestModuleProvide(t *testing.T) {
	t.Parallel()

	c := needle.New()

	module := needle.NewModule("config")
	needle.ModuleProvide(module, func(ctx context.Context, r needle.Resolver) (*Config, error) {
		return &Config{Port: 9000, Host: "module.local"}, nil
	})

	err := c.Apply(module)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	cfg, err := needle.Invoke[*Config](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if cfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Port)
	}
}

func TestModuleProvideValue(t *testing.T) {
	t.Parallel()

	c := needle.New()

	config := &Config{Port: 7000}
	module := needle.NewModule("values")
	needle.ModuleProvideValue(module, config)

	err := c.Apply(module)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	cfg, err := needle.Invoke[*Config](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if cfg != config {
		t.Error("expected same instance")
	}
}

func TestModuleInclude(t *testing.T) {
	t.Parallel()

	c := needle.New()

	configModule := needle.NewModule("config")
	needle.ModuleProvideValue(configModule, &Config{Port: 5000})

	dbModule := needle.NewModule("db")
	needle.ModuleProvide(dbModule, func(ctx context.Context, r needle.Resolver) (*Database, error) {
		cfg := needle.MustInvoke[*Config](c)
		return &Database{Config: cfg, Name: "testdb"}, nil
	})

	appModule := needle.NewModule("app").
		Include(configModule).
		Include(dbModule)

	err := c.Apply(appModule)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	db, err := needle.Invoke[*Database](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if db.Config.Port != 5000 {
		t.Errorf("expected port 5000, got %d", db.Config.Port)
	}
}

func TestModuleBind(t *testing.T) {
	t.Parallel()

	c := needle.New()

	module := needle.NewModule("repos")
	needle.ModuleProvideValue(module, &Database{Name: "postgres"})
	needle.ModuleProvide(module, func(ctx context.Context, r needle.Resolver) (*PostgresUserRepo, error) {
		db := needle.MustInvoke[*Database](c)
		return &PostgresUserRepo{DB: db}, nil
	})
	needle.ModuleBind[UserRepository, *PostgresUserRepo](module)

	err := c.Apply(module)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	repo, err := needle.Invoke[UserRepository](c)
	if err != nil {
		t.Fatalf("Invoke UserRepository failed: %v", err)
	}

	result := repo.FindByID(1)
	if result != "user-postgres" {
		t.Errorf("expected 'user-postgres', got %s", result)
	}
}

func TestModuleDecorate(t *testing.T) {
	t.Parallel()

	c := needle.New()

	module := needle.NewModule("logging")
	needle.ModuleProvide(module, func(ctx context.Context, r needle.Resolver) (*Logger, error) {
		return &Logger{Prefix: "app"}, nil
	})
	needle.ModuleDecorate(module, func(ctx context.Context, r needle.Resolver, base *Logger) (*Logger, error) {
		base.Prefix = "[" + base.Prefix + "]"
		return base, nil
	})

	err := c.Apply(module)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	logger, err := needle.Invoke[*Logger](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if logger.Prefix != "[app]" {
		t.Errorf("expected prefix '[app]', got %s", logger.Prefix)
	}
}

func TestBind(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideValue(c, &Database{Name: "main"})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	err = needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*PostgresUserRepo, error) {
		db := needle.MustInvoke[*Database](c)
		return &PostgresUserRepo{DB: db}, nil
	})
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	err = needle.Bind[UserRepository, *PostgresUserRepo](c)
	if err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	repo, err := needle.Invoke[UserRepository](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if repo.FindByID(1) != "user-main" {
		t.Error("expected 'user-main'")
	}
}

func TestBindNamed(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideValue(c, &Database{Name: "named-db"})
	if err != nil {
		t.Fatalf("ProvideValue failed: %v", err)
	}

	err = needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*PostgresUserRepo, error) {
		db := needle.MustInvoke[*Database](c)
		return &PostgresUserRepo{DB: db}, nil
	})
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	err = needle.BindNamed[UserRepository, *PostgresUserRepo](c, "users")
	if err != nil {
		t.Fatalf("BindNamed failed: %v", err)
	}

	repo, err := needle.InvokeNamed[UserRepository](c, "users")
	if err != nil {
		t.Fatalf("InvokeNamed failed: %v", err)
	}

	if repo.FindByID(1) != "user-named-db" {
		t.Error("expected 'user-named-db'")
	}
}

func TestDecorate(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Logger, error) {
		return &Logger{Prefix: "base"}, nil
	})
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	needle.Decorate(c, func(ctx context.Context, r needle.Resolver, base *Logger) (*Logger, error) {
		base.Prefix = "decorated:" + base.Prefix
		return base, nil
	})

	logger, err := needle.Invoke[*Logger](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if logger.Prefix != "decorated:base" {
		t.Errorf("expected 'decorated:base', got %s", logger.Prefix)
	}
}

func TestDecorateChain(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Logger, error) {
		return &Logger{Prefix: "core"}, nil
	})
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	needle.Decorate(c, func(ctx context.Context, r needle.Resolver, base *Logger) (*Logger, error) {
		base.Prefix = "[1]" + base.Prefix
		return base, nil
	})

	needle.Decorate(c, func(ctx context.Context, r needle.Resolver, base *Logger) (*Logger, error) {
		base.Prefix = "[2]" + base.Prefix
		return base, nil
	})

	logger, err := needle.Invoke[*Logger](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if logger.Prefix != "[2][1]core" {
		t.Errorf("expected '[2][1]core', got %s", logger.Prefix)
	}
}

func TestDecorateNamed(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.ProvideNamed(c, "app", func(ctx context.Context, r needle.Resolver) (*Logger, error) {
		return &Logger{Prefix: "app"}, nil
	})
	if err != nil {
		t.Fatalf("ProvideNamed failed: %v", err)
	}

	needle.DecorateNamed(c, "app", func(ctx context.Context, r needle.Resolver, base *Logger) (*Logger, error) {
		base.Prefix = "named:" + base.Prefix
		return base, nil
	})

	logger, err := needle.InvokeNamed[*Logger](c, "app")
	if err != nil {
		t.Fatalf("InvokeNamed failed: %v", err)
	}

	if logger.Prefix != "named:app" {
		t.Errorf("expected 'named:app', got %s", logger.Prefix)
	}
}

func TestMultipleModules(t *testing.T) {
	t.Parallel()

	c := needle.New()

	configModule := needle.NewModule("config")
	needle.ModuleProvideValue(configModule, &Config{Port: 8080})

	dbModule := needle.NewModule("db")
	needle.ModuleProvide(dbModule, func(ctx context.Context, r needle.Resolver) (*Database, error) {
		cfg := needle.MustInvoke[*Config](c)
		return &Database{Config: cfg, Name: "app-db"}, nil
	})

	err := c.Apply(configModule, dbModule)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	db, err := needle.Invoke[*Database](c)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if db.Config.Port != 8080 {
		t.Errorf("expected port 8080, got %d", db.Config.Port)
	}
}
