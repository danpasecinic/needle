# Needle

A modern, type-safe dependency injection framework for Go 1.25+.

[![Go Reference](https://pkg.go.dev/badge/github.com/danpasecinic/needle.svg)](https://pkg.go.dev/github.com/danpasecinic/needle)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/needle)](https://goreportcard.com/report/github.com/danpasecinic/needle)

## Features

- **Type-safe generics** - Compile-time type checking with `Provide[T]` and `Invoke[T]`
- **Auto-wiring** - Constructor injection and struct tag injection
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

### vs uber/fx

| Operation                | Needle | Fx    | Speedup         |
|--------------------------|--------|-------|-----------------|
| Provide (10 services)    | 18μs   | 209μs | **12x faster**  |
| Start+Stop (10 services) | 15μs   | 27μs  | **1.8x faster** |
| Start+Stop (50 services) | 53μs   | 101μs | **1.9x faster** |
| Memory (10 services)     | 23KB   | 169KB | **7x less**     |

<img width="1605" height="535" alt="image" src="https://github.com/user-attachments/assets/fc6d3b48-d2af-4789-ba94-45a386ab279c" />

### Parallel Startup

When services have initialization work (database connections, HTTP clients, etc.),
parallel startup provides significant speedups:

| Scenario               | Sequential | Parallel | Speedup |
|------------------------|------------|----------|---------|
| 10 services × 1ms work | 23ms       | 2.5ms    | **9x**  |
| 50 services × 1ms work | 117ms      | 2.8ms    | **42x** |


## Documentation

See [pkg.go.dev](https://pkg.go.dev/github.com/danpasecinic/needle) for full API documentation.

## License

MIT License - see [LICENSE](LICENSE) for details.
