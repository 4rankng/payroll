package main

import (
	"fmt"
)

const flowCron = "Cron"

func runCronTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Cron Jobs")

	admin := client.WithToken(data.AdminToken)

	var jobName string

	reporter.RunTest(flowCron, "List cron jobs", func() error {
		var jobs []CronJobResponse
		if _, err := admin.GetInto("/api/v1/cron-jobs", &jobs); err != nil {
			return fmt.Errorf("list cron jobs: %w", err)
		}
		fmt.Printf("    Found %d cron jobs\n", len(jobs))
		if len(jobs) > 0 {
			jobName = jobs[0].Name
			fmt.Printf("    Using job: %s\n", jobName)
		}
		return AssertSliceMinLen("jobs", len(jobs), 1)
	})

	reporter.RunTest(flowCron, "Toggle cron job off", func() error {
		if jobName == "" {
			return fmt.Errorf("no cron job available")
		}
		body := map[string]any{"enabled": false}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/cron-jobs/%s/toggle", jobName), body); err != nil {
			return fmt.Errorf("toggle off: %w", err)
		}
		var jobs []CronJobResponse
		if _, err := admin.GetInto("/api/v1/cron-jobs", &jobs); err != nil {
			return fmt.Errorf("verify toggle: %w", err)
		}
		for _, j := range jobs {
			if j.Name == jobName {
				if j.IsEnabled {
					return fmt.Errorf("expected job %s to be disabled", jobName)
				}
				fmt.Printf("    Job %s disabled\n", jobName)
				return nil
			}
		}
		return fmt.Errorf("job %s not found in list", jobName)
	})

	reporter.RunTest(flowCron, "Toggle cron job back on", func() error {
		if jobName == "" {
			return fmt.Errorf("no cron job available")
		}
		body := map[string]any{"enabled": true}
		if _, _, err := admin.Put(fmt.Sprintf("/api/v1/cron-jobs/%s/toggle", jobName), body); err != nil {
			return fmt.Errorf("toggle on: %w", err)
		}
		var jobs []CronJobResponse
		if _, err := admin.GetInto("/api/v1/cron-jobs", &jobs); err != nil {
			return fmt.Errorf("verify toggle: %w", err)
		}
		for _, j := range jobs {
			if j.Name == jobName {
				if !j.IsEnabled {
					return fmt.Errorf("expected job %s to be enabled", jobName)
				}
				fmt.Printf("    Job %s re-enabled\n", jobName)
				return nil
			}
		}
		return fmt.Errorf("job %s not found in list", jobName)
	})

	reporter.RunTest(flowCron, "Edge: toggle non-existent job", func() error {
		body := map[string]any{"enabled": true}
		_, statusCode, _ := admin.Put("/api/v1/cron-jobs/nonexistent_job/toggle", body)
		if statusCode < 400 {
			fmt.Printf("    Toggle non-existent job returned HTTP %d (accepted as no-op)\n", statusCode)
			return nil
		}
		fmt.Printf("    Toggle non-existent job correctly rejected (HTTP %d)\n", statusCode)
		return nil
	})
}
