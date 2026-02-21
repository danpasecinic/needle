package needle

import (
	"context"
	"sync"
	"time"
)

type ResolveHook func(key string, duration time.Duration, err error)

type ProvideHook func(key string)

type StartHook func(key string, duration time.Duration, err error)

type StopHook func(key string, duration time.Duration, err error)

type HealthStatus string

const (
	HealthStatusUp      HealthStatus = "up"
	HealthStatusDown    HealthStatus = "down"
	HealthStatusUnknown HealthStatus = "unknown"
)

type HealthReport struct {
	Name    string
	Status  HealthStatus
	Error   error
	Latency time.Duration
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type ReadinessChecker interface {
	ReadinessCheck(ctx context.Context) error
}

func (c *Container) Live(ctx context.Context) error {
	reports := c.checkHealth(ctx)
	for _, r := range reports {
		if r.Status == HealthStatusDown {
			return errHealthCheckFailed(r.Name, r.Error)
		}
	}
	return nil
}

func (c *Container) Ready(ctx context.Context) error {
	reports := c.checkReadiness(ctx)
	for _, r := range reports {
		if r.Status == HealthStatusDown {
			return errHealthCheckFailed(r.Name, r.Error)
		}
	}
	return nil
}

func (c *Container) Health(ctx context.Context) []HealthReport {
	return c.checkHealth(ctx)
}

func (c *Container) checkHealth(ctx context.Context) []HealthReport {
	return c.runChecks(ctx, func(instance any) func(context.Context) error {
		if hc, ok := instance.(HealthChecker); ok {
			return hc.HealthCheck
		}
		return nil
	})
}

func (c *Container) checkReadiness(ctx context.Context) []HealthReport {
	return c.runChecks(ctx, func(instance any) func(context.Context) error {
		if rc, ok := instance.(ReadinessChecker); ok {
			return rc.ReadinessCheck
		}
		return nil
	})
}

func (c *Container) runChecks(ctx context.Context, extractCheck func(any) func(context.Context) error) []HealthReport {
	keys := c.internal.Keys()
	var reports []HealthReport
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, key := range keys {
		instance, ok := c.internal.GetInstance(key)
		if !ok {
			continue
		}

		check := extractCheck(instance)
		if check == nil {
			continue
		}

		wg.Add(1)
		go func(k string, fn func(context.Context) error) {
			defer wg.Done()

			start := time.Now()
			err := fn(ctx)
			latency := time.Since(start)

			report := HealthReport{
				Name:    k,
				Latency: latency,
			}

			if err != nil {
				report.Status = HealthStatusDown
				report.Error = err
			} else {
				report.Status = HealthStatusUp
			}

			mu.Lock()
			reports = append(reports, report)
			mu.Unlock()
		}(key, check)
	}

	wg.Wait()
	return reports
}
