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
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
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

	t.Run("should capture execution time", func(t *testing.T) {
		idleForPeriod := 100 * time.Millisecond
		registration := HealthCheckRegistration{
			Name: "test",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
				time.Sleep(idleForPeriod)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout: 2 * time.Second,
		}
		healthCheckService := NewHealthCheckService(registration)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())

		if healthCheckReport.Entries[0].Duration < idleForPeriod {
			t.Errorf("expected >= %v, actual: %v", idleForPeriod, healthCheckReport.Entries[0].Duration)
		}
	})

	t.Run("should cancel health check by timeout and use failure status from registration", func(t *testing.T) {
		idleForPeriod := 1 * time.Hour
		registration := HealthCheckRegistration{
			Name: "test",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
				time.Sleep(idleForPeriod)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       100 * time.Millisecond,
			FailureStatus: Unhealthy,
		}
		healthCheckService := NewHealthCheckService(registration)
		healthCheckReport := healthCheckService.CheckHealth(context.Background())
		reportEntry := healthCheckReport.Entries[0]

		if reportEntry.Duration >= idleForPeriod || reportEntry.Duration < registration.Timeout {
			t.Errorf(
				"expected >= %v and < %v, actual: %v",
				registration.Timeout,
				idleForPeriod,
				reportEntry.Duration,
			)
		}

		if healthCheckReport.Entries[0].Status != Unhealthy {
			t.Errorf("expected: %v, actual: %v", Unhealthy, healthCheckReport.Entries[0].Status)
		}
	})

	t.Run("should not cancel health check when timeout is less than execution time", func(t *testing.T) {
		registration := HealthCheckRegistration{
			Name: "test1",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
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

	t.Run("should cancel health check by timeout on parent context", func(t *testing.T) {
		registration := HealthCheckRegistration{
			Name: "foo",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
				time.Sleep(1 * time.Hour)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       2 * time.Hour,
			FailureStatus: Unhealthy,
		}
		healthCheckService := NewHealthCheckService(registration)
		context, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		healthCheckReport := healthCheckService.CheckHealth(context)

		if healthCheckReport.Entries[0].Status != Unhealthy {
			t.Errorf("expected: %v, actual: %v", Unhealthy, healthCheckReport.Entries[0].Status)
		}
	})

	t.Run("complex case", func(t *testing.T) {
		reg1 := HealthCheckRegistration{ // should be cancelled by timeout specified in registration
			Name: "foo",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
				time.Sleep(1 * time.Hour)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       100 * time.Millisecond,
			FailureStatus: Unhealthy,
		}
		reg2 := HealthCheckRegistration{ // should be Healthy
			Name: "bar",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       1 * time.Minute,
			FailureStatus: Unhealthy,
		}
		reg3 := HealthCheckRegistration{ // should be cancelled by timeout on parent context
			Name: "bar",
			HealthCheck: func(HealthCheckContext) HealthCheckResult {
				time.Sleep(10 * time.Second)
				return HealthCheckResult{Status: Healthy}
			},
			Timeout:       1 * time.Minute,
			FailureStatus: Degraded,
		}
		healthCheckService := NewHealthCheckService(reg1, reg2, reg3)
		context, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
		defer cancel()
		healthCheckReport := healthCheckService.CheckHealth(context)

		if healthCheckReport.Entries[0].Status != Unhealthy {
			t.Errorf("expected: %v, actual: %v", Unhealthy, healthCheckReport.Entries[0].Status)
		}

		if healthCheckReport.Entries[1].Status != Healthy {
			t.Errorf("expected: %v, actual: %v", Healthy, healthCheckReport.Entries[1].Status)
		}

		if healthCheckReport.Entries[2].Status != Degraded {
			t.Errorf("expected: %v, actual: %v", Degraded, healthCheckReport.Entries[2].Status)
		}
	})
}
