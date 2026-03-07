# Needle

A modern, type-safe dependency injection framework for Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/danpasecinic/needle.svg)](https://pkg.go.dev/github.com/danpasecinic/needle)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/needle)](https://goreportcard.com/report/github.com/danpasecinic/needle)

## Features

Needle uses Go generics for compile-time type safety (`Provide[T]`, `Invoke[T]`) and has zero external dependencies.

It supports constructor auto-wiring, struct tag injection, multiple scopes (singleton, transient, request, pooled), and lifecycle hooks that run in dependency order. Services can start in parallel, be lazily initialized, or be replaced at runtime without restarting the container.

You can group providers into modules, bind interfaces to implementations, wrap services with decorators, and resolve optional dependencies with a built-in `Optional[T]` type. Health and readiness checks are supported out of the box.

## Installation

```bash
go get github.com/danpasecinic/needle
```

## Quick Start

```go
c := needle.New()

needle.ProvideValue(c, &Config{Port: 8080})
needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
    return &Server{Config: needle.MustInvoke[*Config](c)}, nil
})

server := needle.MustInvoke[*Server](c)
```

## Examples

See the [examples](examples/) directory:

- [basic](examples/basic/) - Simple dependency chain
- [autowire](examples/autowire/) - Struct-based injection
- [httpserver](examples/httpserver/) - HTTP server with lifecycle
- [modules](examples/modules/) - Modules and interface binding
- [scopes](examples/scopes/) - Singleton, Transient, Request, Pooled
- [decorators](examples/decorators/) - Cross-cutting concerns
- [lazy](examples/lazy/) - Deferred instantiation
- [healthchecks](examples/healthchecks/) - Liveness and readiness probes
- [optional](examples/optional/) - Optional dependencies with fallbacks
- [parallel](examples/parallel/) - Parallel startup/shutdown

## Choosing a Scope

| Scope | Lifetime | Use When |
|-------|----------|----------|
| **Singleton** (default) | One instance for the container lifetime | Stateful services: DB pools, config, caches, loggers |
| **Transient** | New instance every resolution | Stateless handlers, commands, lightweight value objects |
| **Request** | One instance per `WithRequestScope(ctx)` | Per-HTTP-request state: request loggers, auth context, transaction managers |
| **Pooled** | Reusable instances from a fixed-size pool | Expensive-to-create, stateless-between-uses resources: gRPC connections, worker objects |

```go
needle.Provide(c, NewService)                              // Singleton (default)
needle.Provide(c, NewHandler, needle.WithScope(needle.Transient))
needle.Provide(c, NewRequestLogger, needle.WithScope(needle.Request))
needle.Provide(c, NewWorker, needle.WithPoolSize(10))      // Pooled with 10 slots
```

Pooled services must be released by the caller via `c.Release(key, instance)`. If the pool is full, the instance is dropped and a warning is logged.

## Replacing Services

Replace services at runtime without restarting the container. Useful for feature flags, A/B testing, test doubles, or configuration updates.

```go
// Replace with a new value
needle.ReplaceValue(c, &Config{Port: 9090})

// Replace with a new provider
needle.Replace(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
    return &Server{Config: needle.MustInvoke[*Config](c)}, nil
})

// Replace with auto-wired constructor
needle.ReplaceFunc[*Service](c, NewService)

// Replace with struct injection
needle.ReplaceStruct[*Service](c)

// Named variants
needle.ReplaceNamedValue(c, "primary", &Config{Port: 5432})
needle.ReplaceNamed(c, "primary", provider)
```

All Replace functions accept the same options as Provide (`WithScope`, `WithOnStart`, `WithOnStop`, `WithLazy`, `WithPoolSize`). If the service does not exist yet, Replace creates it. If it does exist, the old entry is removed from both the registry and the dependency graph before re-registering.

`Must` variants (`MustReplace`, `MustReplaceValue`, `MustReplaceFunc`, `MustReplaceStruct`) panic on error.

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
