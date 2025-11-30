// Package needle provides a type-safe dependency injection framework for Go 1.25+.
//
// Needle is designed to be simple yet powerful, offering compile-time type safety
// through generics, lifecycle management, scoped dependencies, and modular organization.
//
// # Quick Start
//
// Create a container and register providers:
//
//	c := needle.New()
//
//	needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Config, error) {
//	    return &Config{Port: 8080}, nil
//	})
//
//	needle.Provide(c, func(ctx context.Context, r needle.Resolver) (*Server, error) {
//	    cfg := needle.MustInvoke[*Config](c)
//	    return &Server{config: cfg}, nil
//	})
//
//	c.Run()
//
// # Providers
//
// Providers are functions that create instances of a type. They receive a context
// and a Resolver for accessing other dependencies:
//
//	needle.Provide[T](c, provider)           // Register a provider
//	needle.ProvideValue[T](c, value)         // Register an existing value
//	needle.ProvideNamed[T](c, "name", prov)  // Register a named provider
//
// # Resolution
//
// Resolve dependencies using the Invoke functions:
//
//	svc, err := needle.Invoke[*Service](c)   // Returns value and error
//	svc := needle.MustInvoke[*Service](c)    // Panics on error
//
// # Lifecycle
//
// Services can participate in the container's lifecycle:
//
//	type Server struct{}
//	func (s *Server) Start(ctx context.Context) error { ... }
//	func (s *Server) Stop(ctx context.Context) error { ... }
//
// Or use explicit hooks:
//
//	lc := needle.GetLifecycle(r)
//	lc.OnStart(func(ctx context.Context) error { ... })
//	lc.OnStop(func(ctx context.Context) error { ... })
//
// # Modules
//
// Group related providers into modules:
//
//	var ConfigModule = needle.NewModule("config")
//	needle.ModuleProvideValue(ConfigModule, &Config{Port: 8080})
//
//	var HTTPModule = needle.NewModule("http")
//	needle.ModuleProvide(HTTPModule, NewServer)
//	needle.ModuleProvide(HTTPModule, NewRouter)
//
//	c.Apply(ConfigModule, HTTPModule)
//
// Modules can include other modules:
//
//	var AppModule = needle.NewModule("app").
//	    Include(ConfigModule).
//	    Include(HTTPModule)
//
// # Interface Binding
//
// Bind interfaces to concrete implementations:
//
//	needle.Bind[UserRepository, *PostgresUserRepo](c)
//	needle.BindNamed[Cache, *RedisCache](c, "session")
//
// Or within modules:
//
//	needle.ModuleBind[UserRepository, *PostgresUserRepo](module)
//
// # Decorators
//
// Wrap services with cross-cutting concerns:
//
//	needle.Decorate(c, func(ctx context.Context, r needle.Resolver, log *Logger) (*Logger, error) {
//	    return log.Named("app"), nil
//	})
//
// Decorators are applied in order and can be chained:
//
//	needle.Decorate(c, addMetrics)
//	needle.Decorate(c, addTracing)
//
// # Scopes
//
// Control instance lifetime with scopes:
//
//	needle.Provide(c, NewService, needle.WithScope(needle.Transient))
//
// Available scopes: Singleton (default), Transient, Request, Pooled.
package needle
