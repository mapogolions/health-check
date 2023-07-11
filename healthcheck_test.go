package healthcheck

import (
	"context"
	"testing"
	"time"
)

func TestHealthCheckServic(t *testing.T) {
	t.Run("should create report entry", func(t *testing.T) {
		registration := HealthCheckRegistration{
			Name: "api-health-check",
			HealthCheck: func(ctx context.Context, hcc HealthCheckContext) HealthCheckResult {
				return HealthCheckResult{Status: Healthy}
			},
			Timeout: 2 * time.Second,
		}
		healthCheckService := NewHealthCheckService(registration)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())
		if healthCheckReport.Entries[0].Status != Healthy {
			t.Errorf("Expected 1")
		}
	})
}
