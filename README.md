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
- **Lazy providers** - Defer instantiation until first use
- **Shutdown timeout** - Configurable deadline for graceful shutdown
- **Debug visualization** - Print dependency graphs in ASCII or DOT format
- **Health checks** - Liveness and readiness probes for Kubernetes
- **Metrics observers** - Hook into resolve, provide, start, stop operations
- **Optional dependencies** - Type-safe optional resolution with `Optional[T]`

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
    c := needle.New()

    needle.ProvideValue(c, &Config{Port: 8080})

    needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
        cfg := needle.MustInvoke[*Config](c)
        return &Server{Config: cfg}, nil
    })

    server := needle.MustInvoke[*Server](c)
    fmt.Printf("Server configured on port %d\n", server.Config.Port)
}
```

## API Reference

### Container

```go
c := needle.New()
c := needle.New(needle.WithLogger(slog.Default()))
c := needle.New(needle.WithShutdownTimeout(30 * time.Second))

err := c.Validate()
```

### Registering Providers

```go
needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*MyService, error) {
    return &MyService{}, nil
})

needle.ProvideValue(c, &Config{Port: 8080})

needle.ProvideNamed(c, "primary", func(ctx context.Context, r needle.Resolver) (*DB, error) {
    return &DB{Host: "primary.db"}, nil
})

needle.ProvideNamed(c, "replica", func(ctx context.Context, r needle.Resolver) (*DB, error) {
    return &DB{Host: "replica.db"}, nil
})
```

### Resolving Dependencies

```go
svc, err := needle.Invoke[*MyService](c)

svc := needle.MustInvoke[*MyService](c)

db, err := needle.InvokeNamed[*DB](c, "primary")
db := needle.MustInvokeNamed[*DB](c, "replica")

if needle.Has[*Config](c) {
    // ...
}

svc, ok := needle.TryInvoke[*MyService](c)

opt := needle.InvokeOptional[*Cache](c)
if opt.Present() {
    cache := opt.Value()
}
```

### Optional Dependencies

```go
opt := needle.InvokeOptional[*Cache](c)

if opt.Present() {
    cache := opt.Value()
}

cache, ok := opt.Get()

cache := needle.InvokeOptional[*Cache](c).OrElse(&DefaultCache{})

cache := needle.InvokeOptional[*Cache](c).OrElseFunc(func() *Cache {
    return NewExpensiveCache()
})

cache := needle.InvokeOptionalNamed[*Cache](c, "redis").OrElse(nil)

needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*UserService, error) {
    cache := needle.InvokeOptional[*Cache](c).OrElse(nil)
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

err := c.Start(ctx)

err := c.Stop(ctx)

err := c.Run(ctx)
```

### Lazy Providers

Lazy providers defer instantiation until first use:

```go
needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*ExpensiveService, error) {
    return NewExpensiveService(), nil
}, needle.WithLazy())

c.Start(ctx)

svc := needle.MustInvoke[*ExpensiveService](c)
```

With lazy providers:
- The service is NOT instantiated during `Start()`
- Instantiation happens on first `Invoke()`
- `OnStart` hooks run after first instantiation (if container is running)
- `OnStop` hooks still run during `Stop()` for instantiated services
- If never invoked, no instantiation or lifecycle hooks occur

### Shutdown Timeout

Configure a deadline for graceful shutdown:

```go
c := needle.New(needle.WithShutdownTimeout(30 * time.Second))

needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
    return &Server{}, nil
},
    needle.WithOnStop(func(ctx context.Context) error {
        select {
        case <-time.After(60 * time.Second):
            return nil
        case <-ctx.Done():
            return ctx.Err()
        }
    }),
)

c.Start(ctx)
err := c.Stop(ctx)
```

### Debug Visualization

Print the dependency graph for debugging:

```go
c.PrintGraph()

output := c.SprintGraph()

var buf bytes.Buffer
c.FprintGraph(&buf)
```

Output format:
```
● *main.Config
○ *main.Database ← *main.Config
○ *main.Server ← *main.Database
```
- `●` = instantiated
- `○` = not instantiated
- `←` = depends on

Generate Graphviz DOT format:

```go
c.PrintGraphDOT()

output := c.SprintGraphDOT()

var buf bytes.Buffer
c.FprintGraphDOT(&buf)
```

Get structured graph info:

```go
info := c.Graph()
for _, svc := range info.Services {
    fmt.Printf("%s: deps=%v, instantiated=%v\n",
        svc.Key, svc.Dependencies, svc.Instantiated)
}
```

### Scopes

```go
needle.Provide(c, provider)
needle.Provide(c, provider, needle.WithScope(needle.Singleton))

needle.Provide(c, provider, needle.WithScope(needle.Transient))

needle.Provide(c, provider, needle.WithScope(needle.Request))

ctx := needle.WithRequestScope(context.Background())
svc, _ := needle.InvokeCtx[*MyService](ctx, c)

needle.Provide(c, provider, needle.WithPoolSize(10))

c.Release("*mypackage.MyService", instance)
```

### Modules

```go
var ConfigModule = needle.NewModule("config")
needle.ModuleProvideValue(ConfigModule, &Config{Port: 8080})

var DBModule = needle.NewModule("db")
needle.ModuleProvide(DBModule, func(ctx context.Context, r needle.Resolver) (*Database, error) {
    cfg := needle.MustInvoke[*Config](c)
    return &Database{Config: cfg}, nil
})

c.Apply(ConfigModule, DBModule)

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

needle.Bind[UserRepository, *PostgresUserRepo](c)

repo, _ := needle.Invoke[UserRepository](c)

needle.BindNamed[Cache, *RedisCache](c, "session")
cache, _ := needle.InvokeNamed[Cache](c, "session")

needle.ModuleBind[UserRepository, *PostgresUserRepo](module)
```

### Decorators

```go
needle.Decorate(c, func(ctx context.Context, r needle.Resolver, log *Logger) (*Logger, error) {
    return log.Named("app"), nil
})

needle.Decorate(c, addMetrics)
needle.Decorate(c, addTracing)

needle.DecorateNamed(c, "app", func(ctx context.Context, r needle.Resolver, log *Logger) (*Logger, error) {
    return log.WithField("env", "production"), nil
})

needle.ModuleDecorate(module, func(ctx context.Context, r needle.Resolver, svc *MyService) (*MyService, error) {
    return &DecoratedService{base: svc}, nil
})
```

### Health Checks

```go
type Database struct {
    conn *sql.DB
}

func (d *Database) HealthCheck(ctx context.Context) error {
    return d.conn.PingContext(ctx)
}

func (d *Database) ReadinessCheck(ctx context.Context) error {
    return d.conn.PingContext(ctx)
}

err := c.Live(ctx)
err := c.Ready(ctx)
reports := c.Health(ctx)

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

## Examples

See the [examples](examples/) directory for complete working examples:

- [basic](examples/basic/) - Simple dependency chain
- [httpserver](examples/httpserver/) - HTTP server with lifecycle hooks
- [modules](examples/modules/) - Modules and interface binding

## License

MIT License - see [LICENSE](LICENSE) for details.
