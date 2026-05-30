package domain

import (
	"strings"
	"testing"
	"time"
)

func TestEmailAddress_String(t *testing.T) {
	tests := []struct {
		name string
		addr EmailAddress
		want string
	}{
		{
			name: "address with name",
			addr: EmailAddress{Name: "John Doe", Address: "john@example.com"},
			want: "John Doe <john@example.com>",
		},
		{
			name: "address without name",
			addr: EmailAddress{Address: "john@example.com"},
			want: "john@example.com",
		},
		{
			name: "empty name",
			addr: EmailAddress{Name: "", Address: "john@example.com"},
			want: "john@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.String(); got != tt.want {
				t.Errorf("EmailAddress.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailAddress_Mask(t *testing.T) {
	tests := []struct {
		name string
		addr EmailAddress
		want string
	}{
		{
			name: "normal email",
			addr: EmailAddress{Address: "john@example.com"},
			want: "j***@example.com",
		},
		{
			name: "short email",
			addr: EmailAddress{Address: "a@example.com"},
			want: "a***@example.com",
		},
		{
			name: "email with unicode",
			addr: EmailAddress{Address: "ñoño@example.com"},
			want: "ñ***@example.com",
		},
		{
			name: "invalid email without @",
			addr: EmailAddress{Address: "notanemail"},
			want: "***",
		},
		{
			name: "email with empty local part",
			addr: EmailAddress{Address: "@example.com"},
			want: "***@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.addr.Mask(); got != tt.want {
				t.Errorf("EmailAddress.Mask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEmailAddress(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    EmailAddress
		wantErr bool
	}{
		{
			name:    "simple email",
			raw:     "john@example.com",
			want:    EmailAddress{Name: "", Address: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "email with name",
			raw:     "John Doe <john@example.com>",
			want:    EmailAddress{Name: "John Doe", Address: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "email with whitespace",
			raw:     "  john@example.com  ",
			want:    EmailAddress{Name: "", Address: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "email with uppercase",
			raw:     "John@EXAMPLE.COM",
			want:    EmailAddress{Name: "", Address: "john@example.com"},
			wantErr: false,
		},
		{
			name:    "invalid email",
			raw:     "notanemail",
			wantErr: true,
		},
		{
			name:    "empty email",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "email without domain",
			raw:     "john@",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEmailAddress(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEmailAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Name != tt.want.Name || got.Address != tt.want.Address) {
				t.Errorf("ParseEmailAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailAttachment_Size(t *testing.T) {
	tests := []struct {
		name       string
		attachment EmailAttachment
		want       int
	}{
		{
			name: "small attachment",
			attachment: EmailAttachment{
				Filename: "test.txt",
				Content:  []byte("Hello, World!"),
			},
			want: 13,
		},
		{
			name: "empty attachment",
			attachment: EmailAttachment{
				Filename: "empty.txt",
				Content:  []byte{},
			},
			want: 0,
		},
		{
			name: "large attachment",
			attachment: EmailAttachment{
				Filename: "large.bin",
				Content:  make([]byte, 1024*1024),
			},
			want: 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.attachment.Size(); got != tt.want {
				t.Errorf("EmailAttachment.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailMessage_Validate(t *testing.T) {
	validEmail := EmailMessage{
		To:       []EmailAddress{{Address: "test@example.com"}},
		Subject:  "Test Subject",
		HTMLBody: "<p>Test body</p>",
	}

	tests := []struct {
		name    string
		msg     EmailMessage
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid message",
			msg:     validEmail,
			wantErr: false,
		},
		{
			name: "no recipients",
			msg: EmailMessage{
				Subject:  "Test",
				HTMLBody: "Body",
			},
			wantErr: true,
			errMsg:  "at least one recipient",
		},
		{
			name: "too many recipients",
			msg: EmailMessage{
				To:       make([]EmailAddress, 51),
				Subject:  "Test",
				HTMLBody: "Body",
			},
			wantErr: true,
			errMsg:  "recipient list exceeds",
		},
		{
			name: "too many cc",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				CC:       make([]EmailAddress, 51),
				Subject:  "Test",
				HTMLBody: "Body",
			},
			wantErr: true,
			errMsg:  "cc list exceeds",
		},
		{
			name: "too many bcc",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				BCC:      make([]EmailAddress, 51),
				Subject:  "Test",
				HTMLBody: "Body",
			},
			wantErr: true,
			errMsg:  "bcc list exceeds",
		},
		{
			name: "empty subject",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "",
				HTMLBody: "Body",
			},
			wantErr: true,
			errMsg:  "subject is required",
		},
		{
			name: "subject too long",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  strings.Repeat("a", 151),
				HTMLBody: "Body",
			},
			wantErr: true,
			errMsg:  "subject exceeds maximum length",
		},
		{
			name: "no body",
			msg: EmailMessage{
				To:      []EmailAddress{{Address: "test@example.com"}},
				Subject: "Test",
			},
			wantErr: true,
			errMsg:  "body is required",
		},
		{
			name: "HTML body too large",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "Test",
				HTMLBody: strings.Repeat("a", 101*1024),
			},
			wantErr: true,
			errMsg:  "HTML body exceeds",
		},
		{
			name: "text body too large",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "Test",
				TextBody: strings.Repeat("a", 101*1024),
			},
			wantErr: true,
			errMsg:  "text body exceeds",
		},
		{
			name: "too many attachments",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "Test",
				HTMLBody: "Body",
				Attachments: []EmailAttachment{
					{Filename: "1.txt", Content: []byte("a")},
					{Filename: "2.txt", Content: []byte("b")},
					{Filename: "3.txt", Content: []byte("c")},
					{Filename: "4.txt", Content: []byte("d")},
					{Filename: "5.txt", Content: []byte("e")},
					{Filename: "6.txt", Content: []byte("f")},
				},
			},
			wantErr: true,
			errMsg:  "too many attachments",
		},
		{
			name: "attachment without filename",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "Test",
				HTMLBody: "Body",
				Attachments: []EmailAttachment{
					{Filename: "", Content: []byte("data")},
				},
			},
			wantErr: true,
			errMsg:  "attachment filename is required",
		},
		{
			name: "empty attachment",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "Test",
				HTMLBody: "Body",
				Attachments: []EmailAttachment{
					{Filename: "empty.txt", Content: []byte{}},
				},
			},
			wantErr: true,
			errMsg:  "is empty",
		},
		{
			name: "attachment too large",
			msg: EmailMessage{
				To:       []EmailAddress{{Address: "test@example.com"}},
				Subject:  "Test",
				HTMLBody: "Body",
				Attachments: []EmailAttachment{
					{Filename: "large.bin", Content: make([]byte, 11*1024*1024)},
				},
			},
			wantErr: true,
			errMsg:  "exceeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EmailMessage.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("EmailMessage.Validate() error = %v, should contain %v", err, tt.errMsg)
			}
		})
	}
}

func TestEmailMessage_Validate_SetsDefaults(t *testing.T) {
	msg := EmailMessage{
		To:       []EmailAddress{{Address: "test@example.com"}},
		Subject:  "Test",
		HTMLBody: "Body",
	}

	err := msg.Validate()
	if err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	if msg.Kind != EmailKindGeneric {
		t.Errorf("Kind was not set to default, got %v", msg.Kind)
	}
	if msg.Metadata == nil {
		t.Error("Metadata was not initialized")
	}
	if msg.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
}

func TestNewEmailSentEvent(t *testing.T) {
	result := &EmailDeliveryResult{
		MessageID: "msg-123",
		Provider:  "test-provider",
		SentAt:    time.Now(),
		Metadata:  map[string]string{"key": "value"},
	}

	msg := &EmailMessage{
		Kind:     EmailKindPayrollReport,
		Subject:  "Test Subject",
		TextBody: "Test Body",
		To:       []EmailAddress{{Address: "test@example.com"}},
	}

	userID := uint(42)

	event := NewEmailSentEvent(result, msg, &userID)

	if event.MessageID != result.MessageID {
		t.Errorf("MessageID = %v, want %v", event.MessageID, result.MessageID)
	}
	if event.Kind != msg.Kind {
		t.Errorf("Kind = %v, want %v", event.Kind, msg.Kind)
	}
	if event.Subject != msg.Subject {
		t.Errorf("Subject = %v, want %v", event.Subject, msg.Subject)
	}
	if event.TextBody != msg.TextBody {
		t.Errorf("TextBody = %v, want %v", event.TextBody, msg.TextBody)
	}
	if event.Provider != result.Provider {
		t.Errorf("Provider = %v, want %v", event.Provider, result.Provider)
	}
	if len(event.Recipients) != len(msg.To) {
		t.Errorf("Recipients length = %v, want %v", len(event.Recipients), len(msg.To))
	}
	if event.InitiatedBy == nil || *event.InitiatedBy != userID {
		t.Errorf("InitiatedBy = %v, want %v", event.InitiatedBy, userID)
	}
}
