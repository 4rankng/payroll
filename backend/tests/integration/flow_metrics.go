package main

import "fmt"

const flowMetrics = "Metrics"

func runMetricsTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Metrics Read-Only")

	admin := client.WithToken(data.AdminToken)

	endpoints := []struct {
		name string
		path string
	}{
		{"API metrics", "/api/v1/metrics/api"},
		{"Event bus metrics", "/api/v1/metrics/event-bus"},
		{"Cache metrics", "/api/v1/metrics/cache"},
		{"Errors overview", "/api/v1/metrics/errors/recent"},
		{"Latency overview", "/api/v1/metrics/latency/trend"},
		{"Top endpoints", "/api/v1/metrics/top-endpoints"},
		{"Failed logins overview", "/api/v1/metrics/failed-logins"},
		{"Browser platform stats", "/api/v1/metrics/browser-platform-stats"},
	}

	for _, ep := range endpoints {
		ep := ep
		reporter.RunTest(flowMetrics, "GET "+ep.name, func() error {
			var resp interface{}
			if _, err := admin.GetInto(ep.path, &resp); err != nil {
				return fmt.Errorf("%s: %w", ep.name, err)
			}
			fmt.Printf("    OK: %s\n", ep.name)
			return nil
		})
	}
}
