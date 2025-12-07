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
//	c.Run(ctx)
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
// # Auto-Wiring
//
// Reduce boilerplate with constructor auto-wiring and struct tag injection.
//
// Constructor auto-wiring automatically resolves function parameters:
//
//	func NewUserService(db *Database, log *Logger) *UserService {
//	    return &UserService{db: db, log: log}
//	}
//	needle.ProvideFunc[*UserService](c, NewUserService)
//
// Struct tag injection uses the `needle` tag to inject fields:
//
//	type UserService struct {
//	    DB     *Database `needle:""`           // inject by type
//	    Log    *Logger   `needle:"appLogger"`  // inject by name
//	    Cache  *Cache    `needle:",optional"`  // optional dependency
//	}
//	needle.ProvideStruct[*UserService](c)
//
// Or invoke directly without registering:
//
//	svc, err := needle.InvokeStruct[*UserService](c)
//
// # Resolution
//
// Resolve dependencies using the Invoke functions:
//
//	svc, err := needle.Invoke[*Service](c)   // Returns value and error
//	svc := needle.MustInvoke[*Service](c)    // Panics on error
//
// # Optional Dependencies
//
// Use Optional for dependencies that may or may not be registered:
//
//	opt := needle.InvokeOptional[*Cache](c)
//	if opt.Present() {
//	    cache := opt.Value()
//	}
//
//	// Or use OrElse for default values
//	cache := needle.InvokeOptional[*Cache](c).OrElse(defaultCache)
//
//	// OrElseFunc for lazy defaults
//	cache := needle.InvokeOptional[*Cache](c).OrElseFunc(func() *Cache {
//	    return NewDefaultCache()
//	})
//
// # Lifecycle
//
// Services can participate in the container's lifecycle:
//
//	needle.Provide(c, NewServer,
//	    needle.WithOnStart(func(ctx context.Context) error {
//	        return server.Listen()
//	    }),
//	    needle.WithOnStop(func(ctx context.Context) error {
//	        return server.Shutdown(ctx)
//	    }),
//	)
//
//	c.Start(ctx)  // Starts all services in dependency order
//	c.Stop(ctx)   // Stops all services in reverse order
//	c.Run(ctx)    // Start + wait for signal + Stop
//
// # Lazy Providers
//
// Defer instantiation until first use:
//
//	needle.Provide(c, NewExpensiveService, needle.WithLazy())
//
// Lazy services are not instantiated during Start(). They are created on first
// Invoke(), and their OnStart hooks run at that time if the container is running.
//
// # Parallel Startup
//
// Start independent services concurrently for faster boot times:
//
//	c := needle.New(needle.WithParallel())
//
// Services at the same dependency level start in parallel. Services still
// wait for their dependencies before starting.
//
// # Shutdown Timeout
//
// Configure a deadline for graceful shutdown:
//
//	c := needle.New(needle.WithShutdownTimeout(30 * time.Second))
//
// The timeout applies to Stop() and is checked between service shutdowns.
// Individual OnStop hooks receive the timeout context.
//
// # Debug Visualization
//
// Print the dependency graph for debugging:
//
//	c.PrintGraph()           // ASCII to stdout
//	c.PrintGraphDOT()        // Graphviz DOT to stdout
//	output := c.SprintGraph()
//	info := c.Graph()        // Structured GraphInfo
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
//	needle.Provide(c, NewService, needle.WithScope(needle.Request))
//	needle.Provide(c, NewService, needle.WithPoolSize(10))
//
// Available scopes: Singleton (default), Transient, Request, Pooled.
//
// # Health Checks
//
// Services can implement health check interfaces:
//
//	type Database struct{}
//	func (d *Database) HealthCheck(ctx context.Context) error { return d.Ping(ctx) }
//	func (d *Database) ReadinessCheck(ctx context.Context) error { return d.Ready(ctx) }
//
// Check health status:
//
//	err := c.Live(ctx)           // Fails if any HealthChecker returns error
//	err := c.Ready(ctx)          // Fails if any ReadinessChecker returns error
//	reports := c.Health(ctx)     // Get detailed health reports with latency
//
// # Metrics Observers
//
// Observe container operations for metrics integration:
//
//	c := needle.New(
//	    needle.WithResolveObserver(func(key string, d time.Duration, err error) {
//	        metrics.RecordResolve(key, d, err)
//	    }),
//	    needle.WithProvideObserver(func(key string) {
//	        metrics.RecordProvide(key)
//	    }),
//	    needle.WithStartObserver(func(key string, d time.Duration, err error) {
//	        metrics.RecordStart(key, d, err)
//	    }),
//	    needle.WithStopObserver(func(key string, d time.Duration, err error) {
//	        metrics.RecordStop(key, d, err)
//	    }),
//	)
package needle
