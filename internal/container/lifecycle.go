package container

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func (c *Container) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateNew && c.state != StateStopped {
		c.mu.Unlock()
		return fmt.Errorf("container already started")
	}
	c.state = StateStarting
	c.mu.Unlock()

	var err error
	if c.parallel {
		err = c.startParallel(ctx)
	} else {
		err = c.startSequential(ctx)
	}

	if err != nil {
		return err
	}

	c.mu.Lock()
	c.state = StateRunning
	c.mu.Unlock()

	return nil
}

func (c *Container) startSequential(ctx context.Context) error {
	order, err := c.graph.StartupOrder()
	if err != nil {
		return fmt.Errorf("failed to determine startup order: %w", err)
	}

	for _, key := range order {
		if err := c.startService(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) startParallel(ctx context.Context) error {
	groups, err := c.graph.ParallelStartupGroups()
	if err != nil {
		return fmt.Errorf("failed to determine startup groups: %w", err)
	}

	for _, group := range groups {
		if err := c.startGroup(ctx, group.Nodes); err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) startGroup(ctx context.Context, keys []string) error {
	if len(keys) == 1 {
		return c.startService(ctx, keys[0])
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(keys))

	for _, key := range keys {
		if c.registry.IsLazy(key) {
			continue
		}

		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			if err := c.startService(ctx, k); err != nil {
				errCh <- err
			}
		}(key)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) startService(ctx context.Context, key string) error {
	if c.registry.IsLazy(key) {
		return nil
	}

	start := time.Now()

	if _, err := c.Resolve(ctx, key); err != nil {
		c.callStartHooks(key, time.Since(start), err)
		return fmt.Errorf("failed to resolve %s during startup: %w", key, err)
	}

	entry, exists := c.registry.GetEntry(key)
	if !exists {
		return nil
	}

	var startErr error
	for _, hook := range entry.OnStart {
		c.logger.Debug("running OnStart hook", "service", key)
		if err := hook(ctx); err != nil {
			startErr = fmt.Errorf("OnStart hook failed for %s: %w", key, err)
			break
		}
	}

	c.registry.SetStartRan(key)
	c.callStartHooks(key, time.Since(start), startErr)
	return startErr
}

func (c *Container) callStartHooks(key string, duration time.Duration, err error) {
	for _, hook := range c.onStart {
		hook(key, duration, err)
	}
}

func (c *Container) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateRunning {
		c.mu.Unlock()
		return nil
	}
	c.state = StateStopping
	c.mu.Unlock()

	var errs []error
	if c.parallel {
		errs = c.stopParallel(ctx)
	} else {
		errs = c.stopSequential(ctx)
	}

	c.mu.Lock()
	c.state = StateStopped
	c.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

func (c *Container) stopSequential(ctx context.Context) []error {
	order, err := c.graph.ShutdownOrder()
	if err != nil {
		return []error{fmt.Errorf("failed to determine shutdown order: %w", err)}
	}

	var errs []error
	for _, key := range order {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("shutdown timeout exceeded: %w", err))
			break
		}
		if stopErr := c.stopService(ctx, key); stopErr != nil {
			errs = append(errs, stopErr)
		}
	}

	return errs
}

func (c *Container) stopParallel(ctx context.Context) []error {
	groups, err := c.graph.ParallelShutdownGroups()
	if err != nil {
		return []error{fmt.Errorf("failed to determine shutdown groups: %w", err)}
	}

	var allErrs []error
	for _, group := range groups {
		if err := ctx.Err(); err != nil {
			allErrs = append(allErrs, fmt.Errorf("shutdown timeout exceeded: %w", err))
			break
		}
		errs := c.stopGroup(ctx, group.Nodes)
		allErrs = append(allErrs, errs...)
	}

	return allErrs
}

func (c *Container) stopGroup(ctx context.Context, keys []string) []error {
	if len(keys) == 1 {
		if err := c.stopService(ctx, keys[0]); err != nil {
			return []error{err}
		}
		return nil
	}

	var mu sync.Mutex
	var errs []error
	var wg sync.WaitGroup

	for _, key := range keys {
		entry, exists := c.registry.GetEntry(key)
		if !exists || !entry.Instantiated {
			continue
		}

		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			if err := c.stopService(ctx, k); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(key)
	}

	wg.Wait()
	return errs
}

func (c *Container) stopService(ctx context.Context, key string) error {
	entry, exists := c.registry.GetEntry(key)
	if !exists || !entry.Instantiated {
		return nil
	}

	start := time.Now()
	var stopErr error

	for i := len(entry.OnStop) - 1; i >= 0; i-- {
		c.logger.Debug("running OnStop hook", "service", key)
		if err := entry.OnStop[i](ctx); err != nil {
			stopErr = fmt.Errorf("OnStop hook failed for %s: %w", key, err)
		}
	}

	c.callStopHooks(key, time.Since(start), stopErr)
	return stopErr
}

func (c *Container) callStopHooks(key string, duration time.Duration, err error) {
	for _, hook := range c.onStop {
		hook(key, duration, err)
	}
}
