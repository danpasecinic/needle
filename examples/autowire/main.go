package main

import (
	"fmt"

	"github.com/danpasecinic/needle"
)

type Config struct {
	DatabaseURL string
	CacheSize   int
}

type Logger struct {
	Level string
}

func NewLogger() *Logger {
	return &Logger{Level: "info"}
}

type Database struct {
	URL    string
	Logger *Logger
}

func NewDatabase(cfg *Config, logger *Logger) *Database {
	return &Database{
		URL:    cfg.DatabaseURL,
		Logger: logger,
	}
}

type Cache struct {
	Size int
}

func NewCache(cfg *Config) *Cache {
	return &Cache{Size: cfg.CacheSize}
}

type UserRepository struct {
	DB    *Database `needle:""`
	Cache *Cache    `needle:",optional"`
}

type UserService struct {
	Repo   *UserRepository `needle:""`
	Logger *Logger         `needle:""`
}

func main() {
	c := needle.New()

	_ = needle.ProvideValue(
		c, &Config{
			DatabaseURL: "postgres://localhost/mydb",
			CacheSize:   1000,
		},
	)

	_ = needle.ProvideFunc[*Logger](c, NewLogger)
	_ = needle.ProvideFunc[*Database](c, NewDatabase)
	_ = needle.ProvideFunc[*Cache](c, NewCache)

	_ = needle.ProvideStruct[*UserRepository](c)
	_ = needle.ProvideStruct[*UserService](c)

	if err := c.Validate(); err != nil {
		panic(err)
	}

	svc := needle.MustInvoke[*UserService](c)

	fmt.Println("UserService resolved successfully!")
	fmt.Printf("  Logger level: %s\n", svc.Logger.Level)
	fmt.Printf("  Repo DB URL: %s\n", svc.Repo.DB.URL)
	fmt.Printf("  Repo Cache size: %d\n", svc.Repo.Cache.Size)

	fmt.Println("\n--- Comparison ---")
	fmt.Println("Traditional (verbose):")
	fmt.Println(
		`  needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*UserService, error) {
      repo := needle.MustInvoke[*UserRepository](c)
      logger := needle.MustInvoke[*Logger](c)
      return &UserService{Repo: repo, Logger: logger}, nil
  })`,
	)

	fmt.Println("\nWith ProvideFunc (constructor auto-wiring):")
	fmt.Println(`  needle.ProvideFunc[*Database](c, NewDatabase)`)

	fmt.Println("\nWith ProvideStruct (struct tag injection):")
	fmt.Println(`  needle.ProvideStruct[*UserService](c)`)
}
