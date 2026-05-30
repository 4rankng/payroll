package main

import (
	"fmt"
)

const flowNotification = "Notification"

func runNotificationTests(client *APIClient, data *TestData, reporter *Reporter, cfg *TestConfig) {
	reporter.PrintSection("FLOW: Notifications")

	admin := client.WithToken(data.AdminToken)

	reporter.RunTest(flowNotification, "Get unread count baseline", func() error {
		var resp UnreadCountResponse
		if _, err := admin.GetInto("/api/v1/notifications/unread/count", &resp); err != nil {
			return fmt.Errorf("unread count: %w", err)
		}
		fmt.Printf("    Unread count: %d\n", resp.Count)
		return nil
	})

	reporter.RunTest(flowNotification, "Create custom notification", func() error {
		body := CustomNotificationRequest{
			Title:       "ITest Notification",
			Message:     "Integration test notification body",
			ToAllAdmins: true,
		}
		if _, _, err := admin.Post("/api/v1/notification", body); err != nil {
			return fmt.Errorf("create notification: %w", err)
		}
		fmt.Printf("    Custom notification sent\n")
		return nil
	})

	reporter.RunTest(flowNotification, "List notifications", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/notifications?pageSize=10", &resp); err != nil {
			return fmt.Errorf("list notifications: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowNotification, "Get unread notifications", func() error {
		var resp interface{}
		if _, err := admin.GetInto("/api/v1/notifications/unread", &resp); err != nil {
			return fmt.Errorf("unread notifications: %w", err)
		}
		return nil
	})

	reporter.RunTest(flowNotification, "Mark all as read", func() error {
		if _, _, err := admin.Put("/api/v1/notifications/read-all", nil); err != nil {
			return fmt.Errorf("mark all read: %w", err)
		}
		var countResp UnreadCountResponse
		if _, err := admin.GetInto("/api/v1/notifications/unread/count", &countResp); err != nil {
			return fmt.Errorf("count after mark read: %w", err)
		}
		fmt.Printf("    Unread count after mark-all-read: %d\n", countResp.Count)
		return AssertEqual("count", int64(0), countResp.Count)
	})

	reporter.RunTest(flowNotification, "Edge: create notification with missing title", func() error {
		body := map[string]any{"message": "no title"}
		_, statusCode, err := admin.PostExpectError("/api/v1/notification", body)
		if err != nil {
			return fmt.Errorf("post: %w", err)
		}
		return AssertGreaterOrEqual("status", 400, statusCode)
	})
}
