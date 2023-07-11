package healthcheck

import (
	"context"
	"testing"
	"time"
)

func TestHealthCheckService(t *testing.T) {
	// should cancel N health check by timeout
	t.Run("should cancel health check by timeout and use failure status from registration", func(t *testing.T) {
		idleForPeriod := 60 * time.Second
		registration := HealthCheckRegistration{
			Name: "test",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				time.Sleep(idleForPeriod)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       100 * time.Millisecond,
			FailureStatus: Unhealthy,
		}
		healthCheckService := NewHealthCheckService(registration)
		start := time.Now()
		healthCheckReport := healthCheckService.CheckHealth(context.Background())

		if time.Since(start) >= idleForPeriod {
			t.Errorf("Should be exit by timeout")
		}
		if healthCheckReport.Entries[0].Status != Unhealthy {
			t.Errorf("Should use Unhealthy status from registration")
		}

	})

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
			t.Errorf("Expected Healthy status")
		}
	})
}
