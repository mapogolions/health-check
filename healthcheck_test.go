package healthcheck

import (
	"context"
	"testing"
	"time"
)

func TestHealthCheckService(t *testing.T) {
	t.Run("should run single health check", func(t *testing.T) {
		registration := HealthCheckRegistration{
			Name: "test",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				return HealthCheckResult{Status: Healthy}
			},
			Timeout: 2 * time.Second,
		}
		healthCheckService := NewHealthCheckService(registration)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())
		if healthCheckReport.Entries[0].Status != Healthy {
			t.Errorf("expected: %v, actual: %v", Healthy, healthCheckReport.Entries[0].Status)
		}
	})

	t.Run("should cancel health check by timeout and use failure status from registration", func(t *testing.T) {
		registration := HealthCheckRegistration{
			Name: "test",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				time.Sleep(1 * time.Hour)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       100 * time.Millisecond,
			FailureStatus: Unhealthy,
		}
		healthCheckService := NewHealthCheckService(registration)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())

		if healthCheckReport.Entries[0].Status != Unhealthy {
			t.Errorf("expected: %v, actual: %v", Unhealthy, healthCheckReport.Entries[0].Status)
		}

	})

	t.Run("should cancell N health check by timeout and use failure statuses from registrations", func(t *testing.T) {
		registration1 := HealthCheckRegistration{
			Name: "test1",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				time.Sleep(1 * time.Hour)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       100 * time.Millisecond,
			FailureStatus: Unhealthy,
		}
		registration2 := HealthCheckRegistration{
			Name: "test2",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				time.Sleep(1 * time.Hour)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       50 * time.Millisecond,
			FailureStatus: Degraded,
		}
		healthCheckService := NewHealthCheckService(registration1, registration2)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())

		if healthCheckReport.Entries[0].Status != Unhealthy {
			t.Errorf("expected: %v, actual: %v", Unhealthy, healthCheckReport.Entries[0].Status)
		}
		if healthCheckReport.Entries[1].Status != Degraded {
			t.Errorf("expected: %v, actual: %v", Degraded, healthCheckReport.Entries[1].Status)
		}
	})
}
