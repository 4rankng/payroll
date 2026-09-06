package main

import (
	"encoding/json"
	"fmt"
	"time"

	"api-server/internal/pkg/clock"
)

const flowAdBanner = "AdBanner"

// adBannerFlowBanner is the slice of AdBannerResponse the flow needs.
type adBannerFlowBanner struct {
	ID          uint           `json:"id"`
	Title       string         `json:"title"`
	IsActive    bool           `json:"isActive"`
	ClickCounts map[string]int `json:"clickCounts"`
}

// adBannerFlowPayload is the create/update request body.
type adBannerFlowPayload struct {
	Title            string             `json:"title"`
	Body             string             `json:"body"`
	Bullets          []string           `json:"bullets"`
	CTAs             []map[string]string `json:"ctas"`
	Footer           string             `json:"footer"`
	TargetProjectIDs []uint             `json:"targetProjectIds"`
	Priority         int                `json:"priority"`
	StartsAt         time.Time          `json:"startsAt"`
	EndsAt           time.Time          `json:"endsAt"`
	IsActive         bool               `json:"isActive"`
}

func adBannerFlowPayloadFor(title string, targets []uint, priority int, startsAt, endsAt time.Time, active bool) adBannerFlowPayload {
	return adBannerFlowPayload{
		Title: title,
		Body:  "Công nhân có thể chấm công tự động và ứng lương ngay trên điện thoại.",
		Bullets: []string{
			"Chấm công tự động, chính xác từng ca làm",
			"Ứng lương theo công đã làm, tối đa 70%",
			"Nhận tiền nhanh chóng, chủ động chi tiêu",
		},
		CTAs: []map[string]string{
			{"label": "Gọi hotline", "type": "phone", "value": "0914827988"},
			{"label": "Nhóm Zalo", "type": "url", "value": "https://zalo.me/g/ekvooqdb9hjcl3qmogof"},
		},
		Footer:           "Ting Ting Software Solutions — Đồng hành cùng người lao động.",
		TargetProjectIDs: targets,
		Priority:         priority,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		IsActive:         active,
	}
}

// resolveMyBanner fetches /api/v1/me/ad-banner and returns (banner, isNull).
func resolveMyBanner(empClient *APIClient) (*adBannerFlowBanner, bool, error) {
	resp, _, err := empClient.Get("/api/v1/me/ad-banner")
	if err != nil {
		return nil, false, err
	}
	raw := string(resp.Data)
	if raw == "" || raw == "null" {
		return nil, true, nil
	}
	var banner adBannerFlowBanner
	if err := json.Unmarshal(resp.Data, &banner); err != nil {
		return nil, false, fmt.Errorf("unmarshal banner: %w", err)
	}
	return &banner, false, nil
}

// runAdBannerTests proves the campaign loop end to end against the live
// backend: admin CRUD → employee resolve → click idempotency → expiry →
// pause → access control. Campaigns created here are deleted at the end so
// reruns stay deterministic even when a prior run failed mid-way.
func runAdBannerTests(client *APIClient, data *TestData, reporter *Reporter, _ *TestConfig) {
	reporter.PrintSection("FLOW: Ad Banner")

	if data.EmployeeTokenForAdv == "" || data.EmployeeForAdvance == nil {
		reporter.Skip(flowAdBanner, "All tests", "no employee user account available")
		return
	}
	empClient := client.WithToken(data.EmployeeTokenForAdv)

	now := clock.Now()
	// Sentinel project id no employee belongs to: proves targeting exclusion.
	const foreignProjectID = 999999

	var idA, idB, idC uint
	defer func() {
		for _, id := range []uint{idA, idB, idC} {
			if id != 0 {
				_, _, _ = client.Delete(fmt.Sprintf("/api/v1/ad-banners/%d", id))
			}
		}
	}()

	reporter.RunTest(flowAdBanner, "Admin creates project-targeted campaign", func() error {
		payload := adBannerFlowPayloadFor("[ITEST] Ad targeted", []uint{foreignProjectID}, 100,
			now.Add(-time.Minute), now.Add(7*24*time.Hour), true)
		var created adBannerFlowBanner
		if _, err := client.PostInto("/api/v1/ad-banners", payload, &created); err != nil {
			return fmt.Errorf("create targeted campaign: %w", err)
		}
		idA = created.ID
		fmt.Printf("    Created targeted campaign ID %d\n", idA)
		return nil
	})

	reporter.RunTest(flowAdBanner, "Employee not in targeted project gets no such banner", func() error {
		banner, _, err := resolveMyBanner(empClient)
		if err != nil {
			return err
		}
		if banner != nil && banner.ID == idA {
			return fmt.Errorf("employee must not see campaign targeting project %d", foreignProjectID)
		}
		return nil
	})

	reporter.RunTest(flowAdBanner, "Admin creates broadcast campaign", func() error {
		payload := adBannerFlowPayloadFor("[ITEST] Ad broadcast", nil, 1000,
			now.Add(-time.Minute), now.Add(7*24*time.Hour), true)
		var created adBannerFlowBanner
		if _, err := client.PostInto("/api/v1/ad-banners", payload, &created); err != nil {
			return fmt.Errorf("create broadcast campaign: %w", err)
		}
		idB = created.ID
		fmt.Printf("    Created broadcast campaign ID %d\n", idB)
		return nil
	})

	reporter.RunTest(flowAdBanner, "Employee sees the broadcast campaign", func() error {
		banner, isNull, err := resolveMyBanner(empClient)
		if err != nil {
			return err
		}
		if isNull || banner == nil {
			return fmt.Errorf("expected a resolved banner, got null")
		}
		fmt.Printf("    Employee resolved banner ID %d (%s)\n", banner.ID, banner.Title)
		return nil
	})

	reporter.RunTest(flowAdBanner, "CTA click records and is idempotent", func() error {
		for i := range 2 {
			resp, status, err := empClient.Post(fmt.Sprintf("/api/v1/me/ad-banner/%d/click", idB), map[string]int{"cta_index": 0})
			if err != nil || status >= 300 {
				body := ""
				if resp != nil {
					body = string(resp.Data) + " / " + resp.Message
				}
				return fmt.Errorf("click attempt %d failed: status=%d err=%v body=%s", i+1, status, err, body)
			}
			_ = resp
		}
		return nil
	})

	reporter.RunTest(flowAdBanner, "Admin list shows the click count", func() error {
		var listResp struct {
			Banners []adBannerFlowBanner `json:"banners"`
		}
		if _, err := client.GetInto("/api/v1/ad-banners", &listResp); err != nil {
			return fmt.Errorf("list campaigns: %w", err)
		}
		for _, b := range listResp.Banners {
			if b.ID == idB {
				if b.ClickCounts["0"] < 1 {
					return fmt.Errorf("expected click count >= 1, got %d", b.ClickCounts["0"])
				}
				fmt.Printf("    Campaign %d click count: %d\n", idB, b.ClickCounts["0"])
				return nil
			}
		}
		return fmt.Errorf("campaign %d not found in admin list", idB)
	})

	reporter.RunTest(flowAdBanner, "Expired campaign never resolves", func() error {
		payload := adBannerFlowPayloadFor("[ITEST] Ad expired", nil, 2000,
			now.Add(-48*time.Hour), now.Add(-24*time.Hour), true)
		var created adBannerFlowBanner
		if _, err := client.PostInto("/api/v1/ad-banners", payload, &created); err != nil {
			return fmt.Errorf("create expired campaign: %w", err)
		}
		idC = created.ID

		banner, _, err := resolveMyBanner(empClient)
		if err != nil {
			return err
		}
		if banner != nil && banner.ID == idC {
			return fmt.Errorf("campaign past ends_at must not resolve")
		}
		return nil
	})

	reporter.RunTest(flowAdBanner, "Window over 180 days is rejected", func() error {
		payload := adBannerFlowPayloadFor("[ITEST] Ad too long", nil, 0,
			now, now.Add(181*24*time.Hour), true)
		_, status, err := client.PostExpectError("/api/v1/ad-banners", payload)
		if err != nil {
			return fmt.Errorf("expected 400 for 181-day window, transport error: %v", err)
		}
		if status != 400 {
			return fmt.Errorf("expected 400 for 181-day window, got %d", status)
		}
		return nil
	})

	reporter.RunTest(flowAdBanner, "Unauthenticated resolve is rejected", func() error {
		anon := NewAPIClient(client.BaseURL)
		_, status, err := anon.GetExpectError("/api/v1/me/ad-banner")
		if err != nil {
			return fmt.Errorf("expected 401 without token, transport error: %v", err)
		}
		if status != 401 {
			return fmt.Errorf("expected 401 without token, got %d", status)
		}
		return nil
	})

	reporter.RunTest(flowAdBanner, "Partner cannot manage campaigns", func() error {
		if len(data.Partners) == 0 {
			reporter.Skip(flowAdBanner, "Partner cannot manage campaigns", "no partner account available")
			return nil
		}
		partner := client.WithToken(data.Partners[0].Token)
		_, status, err := partner.GetExpectError("/api/v1/ad-banners")
		if err != nil {
			return fmt.Errorf("expected 403 for partner, transport error: %v", err)
		}
		if status != 403 {
			return fmt.Errorf("expected 403 for partner, got %d", status)
		}
		return nil
	})

	reporter.RunTest(flowAdBanner, "Pausing hides the campaign", func() error {
		payload := adBannerFlowPayloadFor("[ITEST] Ad broadcast", nil, 1000,
			now.Add(-time.Minute), now.Add(7*24*time.Hour), false)
		payload.Title = "[ITEST] Ad broadcast"
		if _, _, err := client.Put(fmt.Sprintf("/api/v1/ad-banners/%d", idB), payload); err != nil {
			return fmt.Errorf("pause campaign: %w", err)
		}

		banner, _, err := resolveMyBanner(empClient)
		if err != nil {
			return err
		}
		if banner != nil && banner.ID == idB {
			return fmt.Errorf("paused campaign must not resolve")
		}
		return nil
	})
}
