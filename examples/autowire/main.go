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

	_ = needle.Register(c, needle.SpecValue(&Config{
		DatabaseURL: "postgres://localhost/mydb",
		CacheSize:   1000,
	}))

	_ = needle.Register(c, needle.SpecFromConstructor[*Logger](NewLogger))
	_ = needle.Register(c, needle.SpecFromConstructor[*Database](NewDatabase))
	_ = needle.Register(c, needle.SpecFromConstructor[*Cache](NewCache))

	_ = needle.Register(c, needle.SpecFromStruct[*UserRepository]())
	_ = needle.Register(c, needle.SpecFromStruct[*UserService]())

	if err := c.Validate(); err != nil {
		panic(err)
	}

	svc := needle.MustInvoke[*UserService](c)

	fmt.Println("UserService resolved successfully!")
	fmt.Printf("  Logger level: %s\n", svc.Logger.Level)
	fmt.Printf("  Repo DB URL: %s\n", svc.Repo.DB.URL)
	fmt.Printf("  Repo Cache size: %d\n", svc.Repo.Cache.Size)

	fmt.Println("\nWith SpecFromConstructor (auto-wired params):")
	fmt.Println(`  needle.Register(c, needle.SpecFromConstructor[*Database](NewDatabase))`)

	fmt.Println("\nWith SpecFromStruct (struct-tag injection):")
	fmt.Println(`  needle.Register(c, needle.SpecFromStruct[*UserService]())`)
}
