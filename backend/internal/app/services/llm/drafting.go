package llm

import (
	"context"
	"fmt"
	"strings"

	llmport "api-server/internal/infra/llm"
)

// draftModel is the flash default from the model-selection spike (2026-10-11);
// the admin "deliverable" route setting (mimo.model_deliverable) overrides it.
const draftModel = "mimo-v2.6-flash"

// modelForDraft resolves the deliverable-route model per call (empty = default).
func (s *Service) modelForDraft(ctx context.Context) string {
	if s.settings == nil {
		return draftModel
	}
	if m := s.settings.GetMimoModelForRoute(ctx, "deliverable"); m != "" {
		return m
	}
	return draftModel
}

// SaoKeDraftFacts is the frozen fact sheet a draft must be built from. Every
// number appears verbatim in the output; the model never computes.
type SaoKeDraftFacts struct {
	ReportAtDate   string
	PeriodLabel    string
	TotalAmount    string
	FeePercent     string
	FeeAmount      string
	TotalWithFee   string
	TimesheetCount int
	RecipientCount int
	DueDate        string
}

// DraftSaoKeEmail writes the Vietnamese subject + body for a sao kê email
// from the deterministic fact sheet. guidance carries the admin's rejection
// note on retry runs ("gửi mai, hôm nay dữ liệu chưa đủ" shapes the redraft).
func (s *Service) DraftSaoKeEmail(ctx context.Context, userID uint, facts SaoKeDraftFacts, guidance string) (subject, body string, err error) {
	system := "Bạn soạn email sao kê lương tuần bằng tiếng Việt, lịch sự, ngắn gọn (tối đa 200 từ). " +
		"CHỈ dùng nguyên văn các con số được cung cấp — tuyệt đối không tự tính toán hay thêm số mới. " +
		"Trả về đúng định dạng: dòng 1 là tiêu đề email, dòng trống, phần còn lại là nội dung email. Không thêm chú thích."
	var b strings.Builder
	fmt.Fprintf(&b, "Dữ kiện (nguyên văn):\n- Ngày sao kê: %s\n- Kỳ lương: %s\n- Tổng chi phí: %s\n- Phí dịch vụ: %s\n- Số phí: %s\n- Tổng cộng: %s\n- Số bảng công: %d\n- Số người nhận: %d\n- Hạn kiểm tra: %s\n",
		facts.ReportAtDate, facts.PeriodLabel, facts.TotalAmount, facts.FeePercent, facts.FeeAmount,
		facts.TotalWithFee, facts.TimesheetCount, facts.RecipientCount, facts.DueDate)
	if g := strings.TrimSpace(guidance); g != "" {
		fmt.Fprintf(&b, "\nLưu ý từ người duyệt cho lần soạn này: %s\n", g)
	}
	resp, err := s.Chat(ctx, userID, llmport.ChatRequest{
		Model: s.modelForDraft(ctx),
		Messages: []llmport.Message{
			{Role: "system", Content: []llmport.ContentPart{llmport.TextPart(system)}},
			{Role: "user", Content: []llmport.ContentPart{llmport.TextPart(b.String())}},
		},
		MaxTokens: 600,
	})
	if err != nil {
		return "", "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", "", fmt.Errorf("mimo không trả về nội dung khi soạn email sao kê")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return "", "", fmt.Errorf("mimo trả về nội dung rỗng khi soạn email sao kê")
	}
	parts := strings.SplitN(content, "\n\n", 2)
	subject = strings.TrimSpace(strings.TrimPrefix(parts[0], "Tiêu đề:"))
	if len(parts) < 2 || subject == "" {
		return "", "", fmt.Errorf("định dạng bản thảo không hợp lệ")
	}
	return subject, strings.TrimSpace(parts[1]), nil
}
