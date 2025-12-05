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
- **Modules** - Group related providers into reusable modules
- **Interface binding** - Bind interfaces to concrete implementations
- **Decorators** - Wrap services with cross-cutting concerns
- **Health checks** - Liveness and readiness probes for Kubernetes
- **Metrics observers** - Hook into resolve, provide, start, stop operations
- **Optional dependencies** - Type-safe optional resolution with `Optional[T]`
- **Testing utilities** - Mock containers, test helpers, and assertions

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

// Optional dependencies (returns Optional[T])
opt := needle.InvokeOptional[*Cache](c)
if opt.Present() {
    cache := opt.Value()
}
```

### Optional Dependencies

```go
// InvokeOptional returns Optional[T] instead of error
opt := needle.InvokeOptional[*Cache](c)

// Check if present
if opt.Present() {
    cache := opt.Value()
    // use cache
}

// Get with boolean (like map access)
cache, ok := opt.Get()

// Provide default value if not present
cache := needle.InvokeOptional[*Cache](c).OrElse(&DefaultCache{})

// Lazy default (function only called if not present)
cache := needle.InvokeOptional[*Cache](c).OrElseFunc(func() *Cache {
    return NewExpensiveCache()
})

// Named optional
cache := needle.InvokeOptionalNamed[*Cache](c, "redis").OrElse(nil)

// Use in providers for optional dependencies
needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*UserService, error) {
    // Cache is optional - service works without it
    cache := needle.InvokeOptional[*Cache](c).OrElse(nil)

    // Metrics is optional - use no-op if not configured
    metrics := needle.InvokeOptional[*Metrics](c).OrElseFunc(NewNoOpMetrics)

    return &UserService{
        Cache:   cache,
        Metrics: metrics,
    }, nil
})
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

### Modules

```go
// Create modules to group related providers
var ConfigModule = needle.NewModule("config")
needle.ModuleProvideValue(ConfigModule, &Config{Port: 8080})

var DBModule = needle.NewModule("db")
needle.ModuleProvide(DBModule, func(ctx context.Context, r needle.Resolver) (*Database, error) {
    cfg := needle.MustInvoke[*Config](c)
    return &Database{Config: cfg}, nil
})

// Apply modules to container
c.Apply(ConfigModule, DBModule)

// Modules can include other modules
var AppModule = needle.NewModule("app").
    Include(ConfigModule).
    Include(DBModule)

c.Apply(AppModule)
```

### Interface Binding

```go
type UserRepository interface {
    FindByID(id int) (*User, error)
}

type PostgresUserRepo struct {
    DB *Database
}

func (r *PostgresUserRepo) FindByID(id int) (*User, error) { ... }

// Bind interface to implementation
needle.Bind[UserRepository, *PostgresUserRepo](c)

// Now you can resolve by interface
repo, _ := needle.Invoke[UserRepository](c)

// Named bindings
needle.BindNamed[Cache, *RedisCache](c, "session")
cache, _ := needle.InvokeNamed[Cache](c, "session")

// Within modules
needle.ModuleBind[UserRepository, *PostgresUserRepo](module)
```

### Decorators

```go
// Wrap services with cross-cutting concerns
needle.Decorate(c, func(ctx context.Context, r needle.Resolver, log *Logger) (*Logger, error) {
    return log.Named("app"), nil
})

// Decorators are applied in order (chaining)
needle.Decorate(c, addMetrics)   // Applied first
needle.Decorate(c, addTracing)   // Applied second

// Named decorators
needle.DecorateNamed(c, "app", func(ctx context.Context, r needle.Resolver, log *Logger) (*Logger, error) {
    return log.WithField("env", "production"), nil
})

// Within modules
needle.ModuleDecorate(module, func(ctx context.Context, r needle.Resolver, svc *MyService) (*MyService, error) {
    return &DecoratedService{base: svc}, nil
})
```

### Health Checks

```go
// Implement HealthChecker for liveness probes
type Database struct {
    conn *sql.DB
}

func (d *Database) HealthCheck(ctx context.Context) error {
    return d.conn.PingContext(ctx)
}

// Implement ReadinessChecker for readiness probes
func (d *Database) ReadinessCheck(ctx context.Context) error {
    // Check if database is ready to accept connections
    return d.conn.PingContext(ctx)
}

// Check health status
err := c.Live(ctx)           // Returns error if any service is unhealthy
err := c.Ready(ctx)          // Returns error if any service is not ready
reports := c.Health(ctx)     // Get detailed health reports with latency

// Use with HTTP handlers for Kubernetes probes
http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    if err := c.Live(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

### Metrics Observers

```go
// Hook into container operations for metrics (Prometheus, OpenTelemetry, etc.)
c := needle.New(
    needle.WithResolveObserver(func(key string, duration time.Duration, err error) {
        resolveLatency.WithLabelValues(key).Observe(duration.Seconds())
        if err != nil {
            resolveErrors.WithLabelValues(key).Inc()
        }
    }),
    needle.WithProvideObserver(func(key string) {
        providedServices.WithLabelValues(key).Inc()
    }),
    needle.WithStartObserver(func(key string, duration time.Duration, err error) {
        startLatency.WithLabelValues(key).Observe(duration.Seconds())
    }),
    needle.WithStopObserver(func(key string, duration time.Duration, err error) {
        stopLatency.WithLabelValues(key).Observe(duration.Seconds())
    }),
)
```

### Testing Utilities

```go
import "github.com/danpasecinic/needle/needletest"

func TestUserService(t *testing.T) {
    // Create test container with auto-cleanup
    tc := needletest.New(t)

    // Provide dependencies (fails test on error)
    needletest.MustProvideValue(tc, &Config{DatabaseURL: "test://localhost"})
    needletest.MustProvide(tc, NewDatabase)
    needletest.MustProvide(tc, NewUserRepository)

    // Replace with mock for testing
    mockRepo := &MockUserRepository{
        FindByIDFn: func(id int) (*User, error) {
            return &User{ID: id, Name: "Test User"}, nil
        },
    }
    needletest.Replace[UserRepository](tc, mockRepo)

    // Assert service exists
    needletest.AssertHas[*Config](tc)

    // Invoke with test failure on error
    svc := needletest.MustInvoke[*UserService](tc)

    // Start/stop with test failure on error
    tc.RequireStart(context.Background())
    defer tc.RequireStop(context.Background())

    // Validate dependency graph
    tc.RequireValidate()
}
```

**Available test helpers:**

- `needletest.New(t)` - Create test container with auto-cleanup
- `needletest.MustProvide[T]` / `needletest.MustProvideValue[T]` - Provide or fail test
- `needletest.MustInvoke[T]` / `needletest.MustInvokeNamed[T]` - Invoke or fail test
- `needletest.Replace[T]` / `needletest.ReplaceNamed[T]` - Replace provider with mock
- `needletest.ReplaceProvider[T]` - Replace with custom provider function
- `needletest.AssertHas[T]` / `needletest.AssertNotHas[T]` - Assert service existence
- `tc.RequireStart` / `tc.RequireStop` - Start/stop or fail test
- `tc.RequireValidate` - Validate or fail test

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
