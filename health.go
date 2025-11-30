package needle

import (
	"context"
	"sync"
	"time"
)

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
	reports := c.checkHealth(ctx, true)
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
	return c.checkHealth(ctx, false)
}

func (c *Container) checkHealth(ctx context.Context, failFast bool) []HealthReport {
	keys := c.internal.Keys()
	var reports []HealthReport
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, key := range keys {
		instance, ok := c.internal.GetInstance(key)
		if !ok {
			continue
		}

		checker, ok := instance.(HealthChecker)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(k string, hc HealthChecker) {
			defer wg.Done()

			start := time.Now()
			err := hc.HealthCheck(ctx)
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
		}(key, checker)
	}

	wg.Wait()
	return reports
}

func (c *Container) checkReadiness(ctx context.Context) []HealthReport {
	keys := c.internal.Keys()
	var reports []HealthReport
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, key := range keys {
		instance, ok := c.internal.GetInstance(key)
		if !ok {
			continue
		}

		checker, ok := instance.(ReadinessChecker)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(k string, rc ReadinessChecker) {
			defer wg.Done()

			start := time.Now()
			err := rc.ReadinessCheck(ctx)
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
		}(key, checker)
	}

	wg.Wait()
	return reports
}
