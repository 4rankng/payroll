package zalo

import (
	"context"
	"strings"
	"testing"
)

func TestSandboxSender_CapturesAndLogs(t *testing.T) {
	s := NewSandboxSender(nil)

	res, err := s.Send(context.Background(), "0987654321", "617976", "track-1", map[string]string{
		"otp_code":             "123456",
		"user_fullname":        "Nguyễn Văn A",
		"otp_valid_in_minutes": "10",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if res.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0", res.ErrorCode)
	}
	if !strings.HasPrefix(res.MsgID, "sandbox-") {
		t.Errorf("MsgID = %q, want sandbox-*", res.MsgID)
	}
	if s.Count() != 1 {
		t.Errorf("Count = %d, want 1", s.Count())
	}
	last := s.LastSend()
	if last == nil {
		t.Fatal("LastSend nil")
	}
	if last.Data["otp_code"] != "123456" {
		t.Errorf("captured otp_code = %q", last.Data["otp_code"])
	}
	if last.Phone != "0987654321" {
		t.Errorf("captured phone = %q", last.Phone)
	}
}

func TestSandboxSender_Reset(t *testing.T) {
	s := NewSandboxSender(nil)
	_, _ = s.Send(context.Background(), "01", "t", "", map[string]string{"otp_code": "1"})
	_, _ = s.Send(context.Background(), "02", "t", "", map[string]string{"otp_code": "2"})
	if s.Count() != 2 {
		t.Fatalf("Count = %d, want 2", s.Count())
	}
	s.Reset()
	if s.Count() != 0 {
		t.Errorf("after Reset, Count = %d, want 0", s.Count())
	}
	if s.LastSend() != nil {
		t.Error("LastSend should be nil after Reset")
	}
}
