# Needle

A modern, type-safe dependency injection framework for Go 1.25+.

[![Go Reference](https://pkg.go.dev/badge/github.com/danpasecinic/needle.svg)](https://pkg.go.dev/github.com/danpasecinic/needle)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/needle)](https://goreportcard.com/report/github.com/danpasecinic/needle)

## Features

- **Type-safe generics** - Compile-time type checking with `Provide[T]` and `Invoke[T]`
- **Auto-wiring** - Constructor injection and struct tag injection
- **Hot reload** - Replace services at runtime without restart
- **Zero dependencies** - Only Go standard library
- **Cycle detection** - Automatically detects circular dependencies
- **Multiple scopes** - Singleton, Transient, Request, Pooled
- **Lifecycle management** - OnStart/OnStop hooks with ordering
- **Lazy providers** - Defer instantiation until first use
- **Parallel startup** - Start independent services concurrently
- **Modules** - Group related providers
- **Interface binding** - Bind interfaces to implementations
- **Decorators** - Wrap services with cross-cutting concerns
- **Health checks** - Liveness and readiness probes
- **Optional dependencies** - Type-safe optional resolution

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
