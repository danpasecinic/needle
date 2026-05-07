// Package needle provides a type-safe dependency injection framework for Go 1.25+.
//
// Needle has one registration entry point. Every service registers via Register[T]
// passing a Spec[T] that captures provider, dependencies, scope, hooks, pool size,
// and lazy flag. Constructor helpers (SpecValue, SpecFromConstructor,
// SpecFromStruct, SpecFromBinding) cover common patterns. Decorators are attached
// separately via Decorate.
//
// # Quick Start
//
// Create a container and register services:
//
//	c := needle.New()
//
//	needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
//
//	needle.Register(c, needle.Spec[*Server]{
//	    Provider: func(ctx context.Context, r needle.Resolver) (*Server, error) {
//	        cfg := needle.MustInvoke[*Config](c)
//	        return &Server{config: cfg}, nil
//	    },
//	})
//
//	c.Run(ctx)
//
// # The Spec
//
// Spec[T] is the single configuration object. Zero values mean "singleton, eager,
// no hooks." Set fields directly or chain helpers:
//
//	type Spec[T any] struct {
//	    Name         string
//	    Provider     Provider[T]
//	    Dependencies []string
//	    Scope        Scope
//	    OnStart      Hook
//	    OnStop       Hook
//	    PoolSize     int
//	    Lazy         bool
//	}
//
// Register errors if the key already exists. Replace overwrites or registers if not
// present. MustRegister and MustReplace panic on error.
//
// # Constructor Helpers
//
// SpecValue binds a pre-built value:
//
//	needle.Register(c, needle.SpecValue(&Config{Port: 8080}))
//
// SpecFromConstructor auto-wires from a constructor's parameters:
//
//	func NewUserService(db *Database, log *Logger) *UserService { ... }
//	needle.Register(c, needle.SpecFromConstructor[*UserService](NewUserService))
//
// SpecFromStruct populates a struct via `needle:"..."` tags:
//
//	type UserService struct {
//	    DB    *Database `needle:""`           // inject by type
//	    Log   *Logger   `needle:"appLogger"`  // inject by name
//	    Cache *Cache    `needle:",optional"`  // optional dependency
//	}
//	needle.Register(c, needle.SpecFromStruct[*UserService]())
//
// SpecFromBinding wires an interface to a registered implementation:
//
//	needle.Register(c, needle.SpecFromBinding[UserRepository, *PostgresUserRepo]())
//
// # Resolution
//
// Resolve dependencies using the Invoke functions:
//
//	svc, err := needle.Invoke[*Service](c)   // returns value and error
//	svc := needle.MustInvoke[*Service](c)    // panics on error
//
// # Optional Dependencies
//
// Use Optional for dependencies that may or may not be registered:
//
//	opt, err := needle.InvokeOptional[*Cache](c)
//	if err != nil {
//	    // registered but resolution failed
//	}
//	if opt.Present() {
//	    cache := opt.Value()
//	}
//
//	cache := opt.OrElse(defaultCache)
//	cache := opt.OrElseFunc(func() *Cache { return NewDefaultCache() })
//
// # Lifecycle
//
// Specs can carry OnStart and OnStop hooks:
//
//	needle.Register(c, needle.Spec[*Server]{
//	    Provider: NewServer,
//	    OnStart:  func(ctx context.Context) error { return server.Listen() },
//	    OnStop:   func(ctx context.Context) error { return server.Shutdown(ctx) },
//	})
//
//	c.Start(ctx)  // starts all services in dependency order
//	c.Stop(ctx)   // stops all services in reverse order
//	c.Run(ctx)    // Start + wait for signal + Stop
//
// Multiple hooks compose via Compose, which runs them in order and stops on first error:
//
//	OnStart: needle.Compose(installRoutes, openListener)
//
// # Lazy Specs
//
// Defer instantiation until first use:
//
//	needle.Register(c, needle.SpecFromConstructor[*Expensive](NewExpensive).WithLazy())
//
// Lazy services are not instantiated during Start. They are created on first
// Invoke, and their OnStart hook runs at that time if the container is running.
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
// The timeout applies to Stop and is checked between service shutdowns.
// Individual OnStop hooks receive the timeout context.
//
// # Debug Visualization
//
// Print the dependency graph for debugging:
//
//	c.PrintGraph()           // ASCII to stdout
//	c.PrintGraphDOT()        // Graphviz DOT to stdout
//	output := c.SprintGraph()
//	info := c.Graph()        // structured GraphInfo
//
// # Modules
//
// Group related specs into modules:
//
//	var ConfigModule = needle.NewModule("config")
//	needle.ModuleRegister(ConfigModule, needle.SpecValue(&Config{Port: 8080}))
//
//	var HTTPModule = needle.NewModule("http")
//	needle.ModuleRegister(HTTPModule, needle.SpecFromConstructor[*Server](NewServer))
//	needle.ModuleRegister(HTTPModule, needle.SpecFromConstructor[*Router](NewRouter))
//
//	c.Apply(ConfigModule, HTTPModule)
//
// Modules can include other modules:
//
//	var AppModule = needle.NewModule("app").
//	    Include(ConfigModule).
//	    Include(HTTPModule)
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
// Control instance lifetime with Scope on the spec:
//
//	needle.Register(c, needle.SpecFromConstructor[*Handler](NewHandler).WithScope(needle.Transient))
//	needle.Register(c, needle.SpecFromConstructor[*ReqLog](NewReqLog).WithScope(needle.Request))
//	needle.Register(c, needle.SpecFromConstructor[*Worker](NewWorker).WithPoolSize(10))
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
//	err := c.Live(ctx)           // fails if any HealthChecker returns error
//	err := c.Ready(ctx)          // fails if any ReadinessChecker returns error
//	reports := c.Health(ctx)     // detailed health reports with latency
//
// # Hot Reload / Dynamic Replacement
//
// Replace services at runtime without restarting the container:
//
//	needle.Replace(c, needle.SpecValue(&Config{NewValue: "updated"}))
//	needle.Replace(c, needle.SpecFromConstructor[*Service](NewService))
//	needle.Replace(c, needle.SpecFromStruct[*Service]())
//	needle.Replace(c, needle.SpecValue(&Config{}).WithName("primary"))
//
// Useful for feature flags, A/B testing, or configuration updates.
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
