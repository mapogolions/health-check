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

	t.Run("should not cancel health check when timeout is less than execution time", func(t *testing.T) {
		registration := HealthCheckRegistration{
			Name: "test1",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				time.Sleep(10 * time.Millisecond)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       1 * time.Minute,
			FailureStatus: Unhealthy,
		}
		healthCheckService := NewHealthCheckService(registration)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())

		if healthCheckReport.Entries[0].Status != Healthy {
			t.Errorf("expected: %v, actual: %v", Healthy, healthCheckReport.Entries[0].Status)
		}
	})
}
