package healthcheck

import (
	"context"
	"time"
)

type HealthCheckStatus int

const (
	Unhealthy HealthCheckStatus = 0
	Degraded  HealthCheckStatus = 1
	Healthy   HealthCheckStatus = 2
)

type HealthCheckResult struct {
	Status      HealthCheckStatus
	Description string
	Error       error
	Data        map[string]any
}

type HealthCheckReportEntry struct {
	Duration    time.Duration
	Status      HealthCheckStatus
	Description string
	Error       error
	Data        map[string]any
}

type HealthCheckRegistration struct {
	Name          string
	HealthCheck   HealthCheck
	FailureStatus HealthCheckStatus
	Tags          []string
	Timeout       time.Duration
}

type HealthCheckContext struct {
	Registration HealthCheckRegistration
}

type HealthCheck func(context.Context, HealthCheckContext) HealthCheckResult

type HealthCheckServiceOptions struct {
	Registrations []HealthCheckRegistration
}

type HealthCheckService struct {
	options HealthCheckServiceOptions
}

type HealthCheckReport struct {
	Entries []HealthCheckReportEntry
}

func (service *HealthCheckService) CheckHealth(ctx context.Context) HealthCheckReport {
	size := len(service.options.Registrations)
	ch := make(chan HealthCheckReportEntry, size)
	defer close(ch)
	for _, registration := range service.options.Registrations {
		go func(registration HealthCheckRegistration) {
			healthCheckContext := HealthCheckContext{Registration: registration}
			context, cancel := context.WithTimeout(ctx, registration.Timeout)
			defer cancel()
			start := time.Now()
			result := <-runHealthCheck(context, healthCheckContext)
			duration := time.Since(start)
			ch <- HealthCheckReportEntry{
				Duration:    duration,
				Status:      result.Status,
				Description: result.Description,
				Error:       result.Error,
				Data:        result.Data}

		}(registration)
	}
	reportEntries := make([]HealthCheckReportEntry, 0, size)
	for reportEntry := range ch {
		reportEntries = append(reportEntries, reportEntry)
	}
	return HealthCheckReport{Entries: reportEntries}
}

func runHealthCheck(ctx context.Context, healthCheckCtx HealthCheckContext) <-chan HealthCheckResult {
	ch := make(chan HealthCheckResult)
	registration := healthCheckCtx.Registration
	defer close(ch)
	go func() {
		select {
		case <-ctx.Done():
			ch <- HealthCheckResult{Status: registration.FailureStatus, Error: ctx.Err(), Description: ctx.Err().Error()}

		case ch <- registration.HealthCheck(ctx, healthCheckCtx):
		}
	}()
	return ch
}
