package main

import (
	"context"
	"fmt"

	"github.com/danpasecinic/needle"
)

type Config struct {
	DatabaseURL string
	Port        int
}

type Database struct {
	URL string
}

func NewDatabase(cfg *Config) *Database {
	return &Database{URL: cfg.DatabaseURL}
}

type UserRepository struct {
	db *Database
}

func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(id int) string {
	return fmt.Sprintf("User %d from %s", id, r.db.URL)
}

type UserService struct {
	repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(id int) string {
	return s.repo.FindByID(id)
}

func main() {
	c := needle.New()

	_ = needle.Register(c, needle.SpecValue(&Config{
		DatabaseURL: "postgres://localhost/mydb",
		Port:        8080,
	}))

	_ = needle.Register(c, needle.Spec[*Database]{
		Provider: func(ctx context.Context, c *needle.Container) (*Database, error) {
			cfg := needle.MustInvoke[*Config](c)
			return NewDatabase(cfg), nil
		},
	})

	_ = needle.Register(c, needle.Spec[*UserRepository]{
		Provider: func(ctx context.Context, c *needle.Container) (*UserRepository, error) {
			db := needle.MustInvoke[*Database](c)
			return NewUserRepository(db), nil
		},
	})

	_ = needle.Register(c, needle.Spec[*UserService]{
		Provider: func(ctx context.Context, c *needle.Container) (*UserService, error) {
			repo := needle.MustInvoke[*UserRepository](c)
			return NewUserService(repo), nil
		},
	})

	if err := c.Validate(); err != nil {
		panic(err)
	}

	svc := needle.MustInvoke[*UserService](c)
	fmt.Println(svc.GetUser(42))
}
