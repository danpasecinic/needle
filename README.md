# Needle

A modern, type-safe dependency injection framework for Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/danpasecinic/needle.svg)](https://pkg.go.dev/github.com/danpasecinic/needle)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/needle)](https://goreportcard.com/report/github.com/danpasecinic/needle)

## Features

Needle uses Go generics for compile-time type safety (`Register[T]`, `Invoke[T]`) and has zero external dependencies.

A single `Register[T]` entry point takes a typed `Spec[T]` carrying provider, dependencies, scope, hooks, pool size, and lazy flag. Constructor helpers (`SpecFromConstructor`, `SpecFromStruct`, `SpecFromBinding`, `SpecValue`) cover auto-wiring, struct-tag injection, interface binding, and pre-built values. Lifecycle hooks run in dependency order, services can start in parallel, and any spec can be replaced at runtime.

You can group specs into modules, attach decorators for cross-cutting concerns, and resolve optional dependencies with the built-in `Optional[T]` type. Health and readiness checks are supported out of the box.

## Installation

```bash
go get github.com/danpasecinic/needle
```

## Quick Start

```go
c := needle.New()

needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
needle.Register(c, needle.Spec[*Server]{
    Provider: func(ctx context.Context, c *needle.Container) (*Server, error) {
        return &Server{Config: needle.MustInvoke[*Config](c)}, nil
    },
})

server := needle.MustInvoke[*Server](c)
```

## Examples

See the [examples](examples/) directory:

- [basic](examples/basic/) - Simple dependency chain
- [autowire](examples/autowire/) - Constructor and struct-tag injection
- [httpserver](examples/httpserver/) - HTTP server with lifecycle
- [modules](examples/modules/) - Modules and interface binding
- [scopes](examples/scopes/) - Singleton, Transient, Request, Pooled
- [decorators](examples/decorators/) - Cross-cutting concerns
- [lazy](examples/lazy/) - Deferred instantiation
- [healthchecks](examples/healthchecks/) - Liveness and readiness probes
- [optional](examples/optional/) - Optional dependencies with fallbacks
- [parallel](examples/parallel/) - Parallel startup/shutdown

## The Spec Type

Every registration goes through one type. The defaults (zero values) cover the common case: singleton scope, eager initialization, no hooks.

```go
type Spec[T any] struct {
    Name         string         // optional, for named services
    Provider     Provider[T]    // factory function (mutually exclusive with SpecValue)
    Dependencies []string       // explicit dependency keys
    Scope        Scope          // Singleton (default), Transient, Request, Pooled
    OnStart      Hook           // lifecycle hook on container Start
    OnStop      Hook            // lifecycle hook on container Stop
    PoolSize    int             // pool size when Scope is Pooled
    Lazy        bool            // defer instantiation until first Resolve
}
```

Constructor helpers fill the spec for common patterns:

```go
needle.SpecValue(&Config{Port: 8080})                       // pre-built value
needle.SpecFromConstructor[*Database](NewDatabase)          // auto-wire from func params
needle.SpecFromStruct[*UserService]()                       // auto-wire from `needle:""` tags
needle.SpecFromBinding[UserRepo, *PostgresRepo]()           // bind interface to impl
```

Each helper returns a `Spec[T]` that you can field-tweak or chain via `WithName`, `WithScope`, `WithLazy`, etc:

```go
needle.Register(c, needle.SpecFromConstructor[*Server](NewServer).
    WithName("primary").
    WithLazy())
```

## Choosing a Scope

| Scope | Lifetime | Use When |
|-------|----------|----------|
| **Singleton** (default) | One instance for the container lifetime | Stateful services: DB pools, config, caches, loggers |
| **Transient** | New instance every resolution | Stateless handlers, commands, lightweight value objects |
| **Request** | One instance per `WithRequestScope(ctx)` | Per-HTTP-request state: request loggers, auth context, transaction managers |
| **Pooled** | Reusable instances from a fixed-size pool | Expensive-to-create, stateless-between-uses resources: gRPC connections, worker objects |

```go
needle.Register(c, needle.SpecFromConstructor[*Service](NewService))                                  // Singleton (default)
needle.Register(c, needle.SpecFromConstructor[*Handler](NewHandler).WithScope(needle.Transient))      // Transient
needle.Register(c, needle.SpecFromConstructor[*RequestLogger](NewRequestLogger).WithScope(needle.Request))
needle.Register(c, needle.SpecFromConstructor[*Worker](NewWorker).WithPoolSize(10))                   // Pooled
```

Pooled services must be released by the caller via `c.Release(key, instance)`. If the pool is full, the instance is dropped and a warning is logged.

## Replacing Services

Replace services at runtime without restarting the container. Useful for feature flags, A/B testing, test doubles, or configuration updates.

```go
needle.Replace(c, needle.SpecValue(&Config{Port: 9090}))

needle.Replace(c, needle.Spec[*Server]{
    Provider: func(ctx context.Context, c *needle.Container) (*Server, error) {
        return &Server{Config: needle.MustInvoke[*Config](c)}, nil
    },
})

needle.Replace(c, needle.SpecFromConstructor[*Service](NewService))
needle.Replace(c, needle.SpecFromStruct[*Service]())

needle.Replace(c, needle.SpecValue(&Config{Port: 5432}).WithName("primary"))
```

`Register` errors on a duplicate key. `Replace` overwrites if the key exists, or registers if it does not. The same `Spec[T]` type is the input to both -- only the intent differs. `MustRegister` and `MustReplace` panic on error.

## Multiple Hooks

`Spec[T]` carries one `OnStart` and one `OnStop` Hook. Combine multiple hooks with `Compose`:

```go
needle.Register(c, needle.Spec[*Server]{
    Provider: NewServer,
    OnStart: needle.Compose(installRoutes, openListener),
    OnStop:  needle.Compose(stopGracefully, flushLogs),
})
```

`Compose` runs hooks in argument order and returns the first error.

## Benchmarks

Needle wins benchmark categories against uber/fx, samber/do, and uber/dig.

### Provider Registration

| Framework  | Simple | Chain | Memory (Chain) |
|------------|--------|-------|----------------|
| **Needle** | 698ns  | 1.5μs | 3KB            |
| Do         | 1.8μs  | 4.4μs | 4KB            |
| Dig        | 13μs   | 26μs  | 28KB           |
| Fx         | 39μs   | 78μs  | 70KB           |

Needle is **56x faster** than Fx for provider registration.

### Service Resolution

| Framework  | Singleton | Chain |
|------------|-----------|-------|
| Fx         | 0ns*      | 0ns*  |
| **Needle** | 15ns      | 17ns  |
| Do         | 150ns     | 161ns |
| Dig        | 614ns     | 622ns |

*Fx resolves at startup, not on-demand.

### Parallel Startup

When services have initialization work (database connections, HTTP clients, etc.):

| Scenario          | Sequential | Parallel | Speedup |
|-------------------|------------|----------|---------|
| 10 services × 1ms | 23ms       | 2.3ms    | **10x** |
| 50 services × 1ms | 113ms      | 2.6ms    | **44x** |

Run benchmarks: `cd benchmark && make run`

## Documentation

See [pkg.go.dev](https://pkg.go.dev/github.com/danpasecinic/needle) for full API documentation.

## License

MIT License - see [LICENSE](LICENSE) for details.
