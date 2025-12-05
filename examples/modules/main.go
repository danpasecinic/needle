package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/danpasecinic/needle"
)

type Config struct {
	DatabaseURL string
	CacheURL    string
}

type Database struct {
	url    string
	logger *slog.Logger
}

func (d *Database) Query(q string) string {
	d.logger.Debug("executing query", "query", q)
	return fmt.Sprintf("result from %s", d.url)
}

type Cache struct {
	url string
}

func (c *Cache) Get(key string) string {
	return fmt.Sprintf("cached:%s", key)
}

type UserRepository interface {
	FindByID(id int) string
	FindByEmail(email string) string
}

type PostgresUserRepository struct {
	db    *Database
	cache *Cache
}

func (r *PostgresUserRepository) FindByID(id int) string {
	if cached := r.cache.Get(fmt.Sprintf("user:%d", id)); cached != "" {
		return cached
	}
	return r.db.Query(fmt.Sprintf("SELECT * FROM users WHERE id = %d", id))
}

func (r *PostgresUserRepository) FindByEmail(email string) string {
	return r.db.Query(fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email))
}

type UserService struct {
	repo   UserRepository
	logger *slog.Logger
}

func (s *UserService) GetUser(id int) string {
	s.logger.Info("getting user", "id", id)
	return s.repo.FindByID(id)
}

var ConfigModule = needle.NewModule("config")
var DatabaseModule = needle.NewModule("database")
var CacheModule = needle.NewModule("cache")
var RepositoryModule = needle.NewModule("repository")
var ServiceModule = needle.NewModule("service")

func init() {
	needle.ModuleProvide(
		DatabaseModule, func(ctx context.Context, r needle.Resolver) (*Database, error) {
			cfg, _ := r.Resolve(ctx, "*main.Config")
			logger, _ := r.Resolve(ctx, "*log/slog.Logger")
			return &Database{
				url:    cfg.(*Config).DatabaseURL,
				logger: logger.(*slog.Logger),
			}, nil
		},
	)

	needle.ModuleProvide(
		CacheModule, func(ctx context.Context, r needle.Resolver) (*Cache, error) {
			cfg, _ := r.Resolve(ctx, "*main.Config")
			return &Cache{url: cfg.(*Config).CacheURL}, nil
		},
	)

	needle.ModuleProvide(
		RepositoryModule, func(ctx context.Context, r needle.Resolver) (*PostgresUserRepository, error) {
			db, _ := r.Resolve(ctx, "*main.Database")
			cache, _ := r.Resolve(ctx, "*main.Cache")
			return &PostgresUserRepository{
				db:    db.(*Database),
				cache: cache.(*Cache),
			}, nil
		},
	)

	needle.ModuleBind[UserRepository, *PostgresUserRepository](RepositoryModule)

	needle.ModuleProvide(
		ServiceModule, func(ctx context.Context, r needle.Resolver) (*UserService, error) {
			repo, _ := r.Resolve(ctx, "main.UserRepository")
			logger, _ := r.Resolve(ctx, "*log/slog.Logger")
			return &UserService{
				repo:   repo.(UserRepository),
				logger: logger.(*slog.Logger),
			}, nil
		},
	)
}

var AppModule = needle.NewModule("app").
	Include(ConfigModule).
	Include(DatabaseModule).
	Include(CacheModule).
	Include(RepositoryModule).
	Include(ServiceModule)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	c := needle.New(needle.WithLogger(logger))

	_ = needle.ProvideValue(c, logger)
	needle.ModuleProvideValue(
		ConfigModule, &Config{
			DatabaseURL: "postgres://localhost/mydb",
			CacheURL:    "redis://localhost:6379",
		},
	)

	if err := c.Apply(AppModule); err != nil {
		logger.Error("failed to apply modules", "error", err)
		os.Exit(1)
	}

	if err := c.Validate(); err != nil {
		logger.Error("validation failed", "error", err)
		os.Exit(1)
	}

	svc := needle.MustInvoke[*UserService](c)
	result := svc.GetUser(42)
	fmt.Println(result)
}
