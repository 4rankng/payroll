package timesheet

import (
	"strings"
	"testing"

	"api-server/internal/app/dto"
)

func TestValidateRejectUnpaidRequest(t *testing.T) {
	tests := []struct {
		name    string
		request dto.RejectUnpaidTimesheetsRequest
		wantErr bool
	}{
		{
			name: "valid inclusive range trims reason",
			request: dto.RejectUnpaidTimesheetsRequest{
				ProjectID: 1, FromDate: "2026-07-01", ToDate: "2026-07-31", RejectionReason: "  Sai kỳ công  ",
			},
		},
		{
			name: "invalid from date",
			request: dto.RejectUnpaidTimesheetsRequest{
				ProjectID: 1, FromDate: "01/07/2026", ToDate: "2026-07-31", RejectionReason: "Sai kỳ công",
			},
			wantErr: true,
		},
		{
			name: "reversed range",
			request: dto.RejectUnpaidTimesheetsRequest{
				ProjectID: 1, FromDate: "2026-07-31", ToDate: "2026-07-01", RejectionReason: "Sai kỳ công",
			},
			wantErr: true,
		},
		{
			name: "blank trimmed reason",
			request: dto.RejectUnpaidTimesheetsRequest{
				ProjectID: 1, FromDate: "2026-07-01", ToDate: "2026-07-31", RejectionReason: "   ",
			},
			wantErr: true,
		},
		{
			name: "reason exceeds 500 unicode characters",
			request: dto.RejectUnpaidTimesheetsRequest{
				ProjectID: 1, FromDate: "2026-07-01", ToDate: "2026-07-31", RejectionReason: strings.Repeat("ộ", 501),
			},
			wantErr: true,
		},
		{
			name: "500 unicode characters accepted",
			request: dto.RejectUnpaidTimesheetsRequest{
				ProjectID: 1, FromDate: "2026-07-01", ToDate: "2026-07-31", RejectionReason: strings.Repeat("ộ", 500),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fromDate, toDate, reason, err := validateRejectUnpaidRequest(test.request)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if fromDate.Format("2006-01-02") != test.request.FromDate || toDate.Format("2006-01-02") != test.request.ToDate {
				t.Fatalf("unexpected parsed range: %s - %s", fromDate, toDate)
			}
			expectedReason := strings.TrimSpace(test.request.RejectionReason)
			if reason != expectedReason {
				t.Fatalf("reason = %q, want trimmed reason", reason)
			}
		})
	}
}
