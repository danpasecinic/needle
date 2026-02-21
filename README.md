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

## Benchmarks

Needle wins benchmark categories against uber/fx, samber/do, and uber/dig.

### Provider Registration

| Framework  | Simple | Chain | Memory (Chain) |
|------------|--------|-------|----------------|
| **Needle** | 780ns  | 1.6μs | 3KB            |
| Do         | 1.9μs  | 5.0μs | 4KB            |
| Dig        | 13μs   | 28μs  | 28KB           |
| Fx         | 42μs   | 85μs  | 70KB           |

Needle is **50x faster** than Fx for provider registration.

### Service Resolution

| Framework  | Singleton | Chain |
|------------|-----------|-------|
| Fx         | 0ns*      | 0ns*  |
| **Needle** | 17ns      | 16ns  |
| Do         | 152ns     | 159ns |
| Dig        | 591ns     | 586ns |

*Fx resolves at startup, not on-demand.

### Parallel Startup

When services have initialization work (database connections, HTTP clients, etc.):

| Scenario          | Sequential | Parallel | Speedup |
|-------------------|------------|----------|---------|
| 10 services × 1ms | 23ms       | 2.4ms    | **10x** |
| 50 services × 1ms | 116ms      | 2.5ms    | **45x** |

Run benchmarks: `cd benchmark && make run`

## Documentation

See [pkg.go.dev](https://pkg.go.dev/github.com/danpasecinic/needle) for full API documentation.

## License

MIT License - see [LICENSE](LICENSE) for details.
