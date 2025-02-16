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
	Context      context.Context
	Registration HealthCheckRegistration
}

type HealthCheck func(HealthCheckContext) HealthCheckResult

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
	Entries  []HealthCheckReportEntry
	Duration time.Duration
}

func (service *HealthCheckService) CheckHealth(ctx context.Context) HealthCheckReport {
	size := len(service.options.Registrations)
	ch := make(chan HealthCheckReportEntry, size)
	wg := sync.WaitGroup{}
	wg.Add(size)
	start := time.Now()
	for i, registration := range service.options.Registrations {
		go func(i int, registration HealthCheckRegistration) {
			defer wg.Done()
			newCtx, cancel := context.WithTimeout(ctx, registration.Timeout)
			defer cancel()
			healthCtx := HealthCheckContext{Registration: registration, Context: newCtx}
			start := time.Now()
			result := runHealthCheck(healthCtx)
			ch <- HealthCheckReportEntry{ // DO NOT use `select`. Non-blocking. Buffered channel has sufficient capacity
				order:       i,
				Duration:    time.Since(start),
				Status:      result.Status,
				Description: result.Description,
				Error:       result.Error,
				Data:        result.Data}
		}(i, registration)
	}
	wg.Wait()
	close(ch)
	duration := time.Since(start)
	reportEntries := make([]HealthCheckReportEntry, 0, size)
	for reportEntry := range ch {
		reportEntries = append(reportEntries, reportEntry)
	}
	sort.Sort(byRegistrationOrder(reportEntries))
	return HealthCheckReport{Entries: reportEntries, Duration: duration}
}

// Pessimistic approach to operation cancellation.
// Instead of relying on the user to check `HealthCheckContext.Context` for cancellation,
// run the operation in a separate goroutine to prevent the health check from hanging indefinitely.
// In the worst case, the spawned computation may keep running forever, consuming system resources.
func runHealthCheck(healthCtx HealthCheckContext) HealthCheckResult {
	r := healthCtx.Registration
	select {
	case <-healthCtx.Context.Done():
		return HealthCheckResult{
			Status:      r.FailureStatus,
			Error:       healthCtx.Context.Err(),
			Description: healthCtx.Context.Err().Error()}
	case result := <-r.healthCheckAsync(healthCtx):
		return result
	}
}

func (registration HealthCheckRegistration) healthCheckAsync(healthCtx HealthCheckContext) <-chan HealthCheckResult {
	// buffered channel prevents memory leaks in case of pessimistic cancellation
	ch := make(chan HealthCheckResult, 1)
	go func() {
		defer close(ch)
		ch <- registration.HealthCheck(healthCtx)
	}()
	return ch
}
