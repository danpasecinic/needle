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

func TestModuleRegister(t *testing.T) {
	t.Parallel()

	c := needle.New()

	module := needle.NewModule("config")
	needle.ModuleRegister(module, needle.Spec[*Config]{
		Provider: func(ctx context.Context, c *needle.Container) (*Config, error) {
			return &Config{Port: 9000, Host: "module.local"}, nil
		},
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

func TestModuleRegisterValue(t *testing.T) {
	t.Parallel()

	c := needle.New()

	config := &Config{Port: 7000}
	module := needle.NewModule("values")
	needle.ModuleRegister(module, needle.SpecValue(config))

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
	needle.ModuleRegister(configModule, needle.SpecValue(&Config{Port: 5000}))

	dbModule := needle.NewModule("db")
	needle.ModuleRegister(dbModule, needle.Spec[*Database]{
		Provider: func(ctx context.Context, c *needle.Container) (*Database, error) {
			cfg := needle.MustInvoke[*Config](c)
			return &Database{Config: cfg, Name: "testdb"}, nil
		},
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

func TestModuleBinding(t *testing.T) {
	t.Parallel()

	c := needle.New()

	module := needle.NewModule("repos")
	needle.ModuleRegister(module, needle.SpecValue(&Database{Name: "postgres"}))
	needle.ModuleRegister(module, needle.Spec[*PostgresUserRepo]{
		Provider: func(ctx context.Context, c *needle.Container) (*PostgresUserRepo, error) {
			db := needle.MustInvoke[*Database](c)
			return &PostgresUserRepo{DB: db}, nil
		},
	})
	needle.ModuleRegister(module, needle.SpecFromBinding[UserRepository, *PostgresUserRepo]())

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
	needle.ModuleRegister(module, needle.Spec[*Logger]{
		Provider: func(ctx context.Context, c *needle.Container) (*Logger, error) {
			return &Logger{Prefix: "app"}, nil
		},
	})
	needle.ModuleDecorate(module, func(ctx context.Context, c *needle.Container, base *Logger) (*Logger, error) {
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

func TestSpecFromBinding(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Register(c, needle.SpecValue(&Database{Name: "main"}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = needle.Register(c, needle.Spec[*PostgresUserRepo]{
		Provider: func(ctx context.Context, c *needle.Container) (*PostgresUserRepo, error) {
			db := needle.MustInvoke[*Database](c)
			return &PostgresUserRepo{DB: db}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = needle.Register(c, needle.SpecFromBinding[UserRepository, *PostgresUserRepo]())
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

func TestSpecFromBindingNamed(t *testing.T) {
	t.Parallel()

	c := needle.New()

	err := needle.Register(c, needle.SpecValue(&Database{Name: "named-db"}))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = needle.Register(c, needle.Spec[*PostgresUserRepo]{
		Provider: func(ctx context.Context, c *needle.Container) (*PostgresUserRepo, error) {
			db := needle.MustInvoke[*Database](c)
			return &PostgresUserRepo{DB: db}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = needle.Register(c, needle.SpecFromBinding[UserRepository, *PostgresUserRepo]().WithName("users"))
	if err != nil {
		t.Fatalf("Register binding failed: %v", err)
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

	err := needle.Register(c, needle.Spec[*Logger]{
		Provider: func(ctx context.Context, c *needle.Container) (*Logger, error) {
			return &Logger{Prefix: "base"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	needle.Decorate(c, func(ctx context.Context, c *needle.Container, base *Logger) (*Logger, error) {
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

	err := needle.Register(c, needle.Spec[*Logger]{
		Provider: func(ctx context.Context, c *needle.Container) (*Logger, error) {
			return &Logger{Prefix: "core"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	needle.Decorate(c, func(ctx context.Context, c *needle.Container, base *Logger) (*Logger, error) {
		base.Prefix = "[1]" + base.Prefix
		return base, nil
	})

	needle.Decorate(c, func(ctx context.Context, c *needle.Container, base *Logger) (*Logger, error) {
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

	err := needle.Register(c, needle.Spec[*Logger]{
		Name: "app",
		Provider: func(ctx context.Context, c *needle.Container) (*Logger, error) {
			return &Logger{Prefix: "app"}, nil
		},
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	needle.DecorateNamed(c, "app", func(ctx context.Context, c *needle.Container, base *Logger) (*Logger, error) {
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
	needle.ModuleRegister(configModule, needle.SpecValue(&Config{Port: 8080}))

	dbModule := needle.NewModule("db")
	needle.ModuleRegister(dbModule, needle.Spec[*Database]{
		Provider: func(ctx context.Context, c *needle.Container) (*Database, error) {
			cfg := needle.MustInvoke[*Config](c)
			return &Database{Config: cfg, Name: "app-db"}, nil
		},
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
