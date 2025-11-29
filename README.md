# Needle

A modern, type-safe dependency injection framework for Go 1.25+.

[![Go Reference](https://pkg.go.dev/badge/github.com/danpasecinic/needle.svg)](https://pkg.go.dev/github.com/danpasecinic/needle)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/needle)](https://goreportcard.com/report/github.com/danpasecinic/needle)

## Features

- **Type-safe generics** - Compile-time type checking with `Provide[T]` and `Invoke[T]`
- **Zero dependencies** - Only Go standard library
- **Cycle detection** - Automatically detects circular dependencies
- **Named services** - Register multiple implementations of the same type
- **Singleton by default** - Efficient instance reuse
- **Lifecycle management** - OnStart/OnStop hooks with proper ordering
- **Multiple scopes** - Singleton, Transient, Request, and Pooled

## Installation

```bash
go get github.com/danpasecinic/needle
```

Requires Go 1.25 or later.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/danpasecinic/needle"
)

type Config struct {
    Port int
}

type Server struct {
    Config *Config
}

func main() {
    // Create container
    c := needle.New()

    // Register a value
    needle.ProvideValue(c, &Config{Port: 8080})

    // Register a provider with dependencies
    needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
        cfg := needle.MustInvoke[*Config](c)
        return &Server{Config: cfg}, nil
    })

    // Resolve dependencies
    server := needle.MustInvoke[*Server](c)
    fmt.Printf("Server configured on port %d\n", server.Config.Port)
}
```

## API Reference

### Container

```go
// Create a new container
c := needle.New()
c := needle.New(needle.WithLogger(slog.Default()))

// Validate the dependency graph
err := c.Validate()
```

### Registering Providers

```go
// Register a provider function
needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*MyService, error) {
    return &MyService{}, nil
})

// Register an existing value
needle.ProvideValue(c, &Config{Port: 8080})

// Register with a name (for multiple implementations)
needle.ProvideNamed(c, "primary", func(ctx context.Context, r needle.Resolver) (*DB, error) {
    return &DB{Host: "primary.db"}, nil
})

needle.ProvideNamed(c, "replica", func(ctx context.Context, r needle.Resolver) (*DB, error) {
    return &DB{Host: "replica.db"}, nil
})
```

### Resolving Dependencies

```go
// Resolve with error handling
svc, err := needle.Invoke[*MyService](c)

// Resolve or panic
svc := needle.MustInvoke[*MyService](c)

// Resolve named service
db, err := needle.InvokeNamed[*DB](c, "primary")
db := needle.MustInvokeNamed[*DB](c, "replica")

// Check if service exists
if needle.Has[*Config](c) {
    // ...
}

// Try to resolve (returns false if not found)
svc, ok := needle.TryInvoke[*MyService](c)
```

### With Context

```go
ctx := context.Background()

svc, err := needle.InvokeCtx[*MyService](ctx, c)
svc := needle.MustInvokeCtx[*MyService](ctx, c)
```

### Lifecycle Management

```go
// Register with lifecycle hooks
needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
    return &Server{}, nil
},
    needle.WithOnStart(func(ctx context.Context) error {
        fmt.Println("Server starting...")
        return nil
    }),
    needle.WithOnStop(func(ctx context.Context) error {
        fmt.Println("Server stopping...")
        return nil
    }),
)

// Start all services (in dependency order)
err := c.Start(ctx)

// Stop all services (in reverse dependency order)
err := c.Stop(ctx)

// Or use Run() to start and wait for shutdown signal
err := c.Run(ctx) // Blocks until SIGINT/SIGTERM or context cancellation
```

### Scopes

```go
// Singleton (default) - one instance per container
needle.Provide(c, provider)
needle.Provide(c, provider, needle.WithScope(needle.Singleton))

// Transient - new instance every time
needle.Provide(c, provider, needle.WithScope(needle.Transient))

// Request - one instance per request context
needle.Provide(c, provider, needle.WithScope(needle.Request))

// Use request scope
ctx := needle.WithRequestScope(context.Background())
svc, _ := needle.InvokeCtx[*MyService](ctx, c)

// Pooled - reusable pool of instances
needle.Provide(c, provider, needle.WithPoolSize(10))

// Release back to pool when done
c.Release("*mypackage.MyService", instance)
```

## Dependency Chain Example

```go
type Config struct {
    DatabaseURL string
}

type Database struct {
    Config *Config
}

type UserRepository struct {
    DB *Database
}

type UserService struct {
    Repo *UserRepository
}

func main() {
    c := needle.New()

    // Register all providers
    needle.ProvideValue(c, &Config{DatabaseURL: "postgres://localhost/mydb"})

    needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Database, error) {
        cfg := needle.MustInvoke[*Config](c)
        return &Database{Config: cfg}, nil
    })

    needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*UserRepository, error) {
        db := needle.MustInvoke[*Database](c)
        return &UserRepository{DB: db}, nil
    })

    needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*UserService, error) {
        repo := needle.MustInvoke[*UserRepository](c)
        return &UserService{Repo: repo}, nil
    })

    // Resolve - all dependencies are automatically resolved
    svc := needle.MustInvoke[*UserService](c)
}
```

## License

MIT License - see [LICENSE](LICENSE) for details.
