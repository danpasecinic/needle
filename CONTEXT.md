# Needle Context

Needle is a generic dependency-injection container for Go. The vocabulary below is what the codebase uses; pick these terms over their aliases.

## Language

**Container**:
The owner of registrations, lifecycle, and resolution. The thing users construct with `needle.New()`.
_Avoid_: registry (that's an internal sub-component), context (overloaded with `context.Context`).

**Spec[T]**:
The configuration a caller hands to `Register[T]` to register a service of type `T`. Sum-typed: carries either a `Provider` or a `Value`, plus optional name, dependencies, scope, hooks, pool size, and lazy flag. The user-facing description of a service.
_Avoid_: definition, registration, config, builder.
  
**ServiceEntry**:
The internal runtime state of a registered service: the spec's contents plus `sync.Once`, init error, pool channel, instantiation flag. Lives inside the registry; users never see it.
_Avoid_: service record, entry (when ambiguous).

**Provider**:
A function `func(ctx, Resolver) (T, error)` that constructs an instance of `T`. One of the two things a `Spec[T]` can carry.
_Avoid_: factory, constructor (constructor refers to plain Go constructor functions used by the autowire path).

**Hook**:
A function `func(ctx) error` called at a service's lifecycle transition. A `Spec[T]` carries at most one `OnStart` and one `OnStop`; multiple hooks compose via `needle.Compose(h1, h2)`.
_Avoid_: callback, listener, observer (observer is reserved for container-wide observation hooks like `WithStartObserver`).

**Scope**:
When and how a service instance is reused: Singleton, Transient, Request, Pooled. A property of the spec; resolved differently per scope.
_Avoid_: lifetime (overloaded with lifecycle), strategy.

**Resolver**:
The lookup interface a `Provider` uses to fetch dependencies during construction. In the deepened design this is the Container itself, not a separate adapter.
_Avoid_: locator, injector.

**Module**:
A deferred recorder of typed registrations. `ModuleRegister[T](m, spec)` captures a closure that calls `Register[T]` against the container at `Apply` time. Modules carry no semantics beyond batched registration.
_Avoid_: bundle, package, group.

**Decorator**:
A function that wraps a resolved instance of `T` to add cross-cutting behaviour. Registered separately from specs (cross-cutting; not per-service config).
_Avoid_: middleware, interceptor.

**Binding**:
A spec where the provider resolves another key and returns it as the registered type. Built via `SpecFromBinding[I, T]()`. Lets an interface `I` be served by an implementation `T` already registered under a different key.
_Avoid_: alias, link.

**Observer**:
A container-wide callback fired on `Resolve`/`Provide`/`Start`/`Stop`. Distinct from per-service Hooks: observers see every service, hooks fire on one service.
_Avoid_: listener, hook (hook is reserved for per-service lifecycle).

## Relationships

- A **Container** holds many **ServiceEntries**, one per registered key.
- A **Spec[T]** is the input to `Register`; the **Container** turns it into a **ServiceEntry**.
- A **ServiceEntry** carries at most one **Provider** (or a pre-built value), at most one **OnStart Hook**, at most one **OnStop Hook**, and exactly one **Scope**.
- A **Module** records typed `Spec[T]` closures and replays them against a **Container** at apply time.
- A **Binding** is a **Spec** whose **Provider** delegates to another key's resolution.
- **Decorators** attach to a key independently of the **Spec** for that key; one key can have many decorators.
- **Observers** attach to the **Container**, not to a **Spec**.

## Example dialogue

> **Dev:** "If I want a service to start lazily and run an `OnStart` hook the first time it resolves, what do I put on the **Spec**?"
> **Maintainer:** "Set `Lazy: true` and `OnStart: myHook`. The **Container** holds the **Spec** as a **ServiceEntry**; on first `Resolve` it constructs the instance via the **Provider** and runs the **OnStart Hook** because the container is already in the running state."

> **Dev:** "Can I add two `OnStart` hooks to one service?"
> **Maintainer:** "A **Spec** carries one **Hook** per slot. Compose them with `needle.Compose(h1, h2)` and assign the composite. **Modules** wanting to layer extra behaviour register a **Decorator** instead — that's the cross-cutting path."

## Flagged ambiguities

- "service" was used loosely for both the runtime instance and the registration. Resolved: the registration is a **Spec[T]** (user-facing) or **ServiceEntry** (internal); the runtime instance is just "the resolved value" or "instance."
- "hook" previously named both per-service lifecycle callbacks and container-wide callbacks. Resolved: per-service is **Hook**; container-wide is **Observer**.
