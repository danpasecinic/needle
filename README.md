# Needle

A modern, type-safe dependency injection framework for Go 1.25+.

[![Go Reference](https://pkg.go.dev/badge/github.com/danpasecinic/needle.svg)](https://pkg.go.dev/github.com/danpasecinic/needle)
[![Go Report Card](https://goreportcard.com/badge/github.com/danpasecinic/needle)](https://goreportcard.com/report/github.com/danpasecinic/needle)

## Features

- **Type-safe generics** - Compile-time type checking with `Provide[T]` and `Invoke[T]`
- **Zero dependencies** - Only Go standard library
- **Cycle detection** - Automatically detects circular dependencies
- **Multiple scopes** - Singleton, Transient, Request, Pooled
- **Lifecycle management** - OnStart/OnStop hooks with ordering
- **Lazy providers** - Defer instantiation until first use
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
- [httpserver](examples/httpserver/) - HTTP server with lifecycle
- [modules](examples/modules/) - Modules and interface binding
- [scopes](examples/scopes/) - Singleton, Transient, Request, Pooled
- [decorators](examples/decorators/) - Cross-cutting concerns
- [lazy](examples/lazy/) - Deferred instantiation
- [healthchecks](examples/healthchecks/) - Liveness and readiness probes
- [optional](examples/optional/) - Optional dependencies with fallbacks

## Documentation

See [pkg.go.dev](https://pkg.go.dev/github.com/danpasecinic/needle) for full API documentation.

## License

MIT License - see [LICENSE](LICENSE) for details.
