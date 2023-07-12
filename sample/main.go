package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mapogolions/healthcheck"
)

const (
	slowResponseDuration = 200 * time.Millisecond
)

func main() {
	googleHealthCheck := healthcheck.HealthCheckRegistration{
		Name:          "google-health-check",
		Timeout:       20 * time.Second,
		HealthCheck:   healthCheckBuilder("http://google.com", slowResponseDuration),
		FailureStatus: healthcheck.Unhealthy}

	githubHealthCheck := healthcheck.HealthCheckRegistration{
		Name:          "github-health-check",
		Timeout:       20 * time.Second,
		HealthCheck:   healthCheckBuilder("http://github.com", slowResponseDuration),
		FailureStatus: healthcheck.Unhealthy}

	healthCheckService := healthcheck.NewHealthCheckService(googleHealthCheck, githubHealthCheck)

	http.HandleFunc("/healthcheck", HealthCheckHandler(healthCheckService))
	err := http.ListenAndServe("localhost:8080", nil)

	if err != nil {
		log.Fatal(err)
	}
}

func HealthCheckHandler(healthCheckService *healthcheck.HealthCheckService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthCheckReport := healthCheckService.CheckHealth(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(healthCheckReport)
	}
}

func healthCheckBuilder(url string, slowResponseDuration time.Duration) healthcheck.HealthCheck {
	return func(hcc healthcheck.HealthCheckContext) healthcheck.HealthCheckResult {
		start := time.Now()
		_, err := http.Get(url)
		elapsed := time.Since(start)
		if err != nil {
			return healthcheck.HealthCheckResult{Status: healthcheck.Unhealthy, Error: err, Description: err.Error()}
		}
		if elapsed >= slowResponseDuration {
			return healthcheck.HealthCheckResult{
				Status:      healthcheck.Degraded,
				Description: fmt.Sprintf("Slow response. Elapsed: %v", elapsed)}
		}
		return healthcheck.HealthCheckResult{
			Status:      healthcheck.Degraded,
			Description: fmt.Sprintf("Response. Elapsed: %v", elapsed)}
	}
}
