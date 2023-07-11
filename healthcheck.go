package healthcheck

import (
	"context"
	"sort"
	"sync"
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
	order       int
	Duration    time.Duration
	Status      HealthCheckStatus
	Description string
	Error       error
	Data        map[string]any
}

type byRegistrationOrder []HealthCheckReportEntry

func (a byRegistrationOrder) Len() int           { return len(a) }
func (a byRegistrationOrder) Less(i, j int) bool { return a[i].order < a[j].order }
func (a byRegistrationOrder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

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

func NewHealthCheckService(registrations ...HealthCheckRegistration) *HealthCheckService {
	options := HealthCheckServiceOptions{Registrations: registrations}
	return &HealthCheckService{options: options}
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
	group := sync.WaitGroup{}
	group.Add(size)
	for i, registration := range service.options.Registrations {
		go func(i int, registration HealthCheckRegistration) {
			defer group.Done()
			healthCheckContext := HealthCheckContext{Registration: registration}
			newCtx, cancel := context.WithTimeout(ctx, registration.Timeout)
			defer cancel()
			start := time.Now()
			result := <-runHealthCheck(newCtx, healthCheckContext)
			ch <- HealthCheckReportEntry{
				order:       i,
				Duration:    time.Since(start),
				Status:      result.Status,
				Description: result.Description,
				Error:       result.Error,
				Data:        result.Data}

		}(i, registration)
	}
	group.Wait()
	close(ch)
	reportEntries := make([]HealthCheckReportEntry, 0, size)
	for reportEntry := range ch {
		reportEntries = append(reportEntries, reportEntry)
	}
	sort.Sort(byRegistrationOrder(reportEntries))
	return HealthCheckReport{Entries: reportEntries}
}

func runHealthCheck(ctx context.Context, healthCheckCtx HealthCheckContext) <-chan HealthCheckResult {
	ch := make(chan HealthCheckResult)
	registration := healthCheckCtx.Registration

	go func() {
		defer close(ch)
		select {
		case <-ctx.Done():
			ch <- HealthCheckResult{Status: registration.FailureStatus, Error: ctx.Err(), Description: ctx.Err().Error()}
		case result := <-func() <-chan HealthCheckResult {
			resultStream := make(chan HealthCheckResult)
			go func() {
				defer close(resultStream)
				resultStream <- registration.HealthCheck(ctx, healthCheckCtx)
			}()
			return resultStream
		}():
			ch <- result
		}
	}()

	return ch
}
