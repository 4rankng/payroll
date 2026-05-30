package converter

import (
	"testing"
)

func TestGetFieldLabel(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{
			name:  "fullname field",
			field: "fullname",
			want:  "Tên đầy đủ",
		},
		{
			name:  "email field",
			field: "email",
			want:  "Email",
		},
		{
			name:  "mobile field",
			field: "mobile",
			want:  "Số điện thoại",
		},
		{
			name:  "cccd field",
			field: "cccd",
			want:  "CCCD",
		},
		{
			name:  "date_of_birth field",
			field: "date_of_birth",
			want:  "Ngày sinh",
		},
		{
			name:  "address field",
			field: "address",
			want:  "Địa chỉ",
		},
		{
			name:  "bank_id field",
			field: "bank_id",
			want:  "Ngân hàng",
		},
		{
			name:  "bank_account_name field",
			field: "bank_account_name",
			want:  "Tên tài khoản",
		},
		{
			name:  "bank_account_number field",
			field: "bank_account_number",
			want:  "Số tài khoản",
		},
		{
			name:  "status field",
			field: "status",
			want:  "Trạng thái",
		},
		{
			name:  "name field",
			field: "name",
			want:  "Tên",
		},
		{
			name:  "code field",
			field: "code",
			want:  "Mã",
		},
		{
			name:  "description field",
			field: "description",
			want:  "Mô tả",
		},
		{
			name:  "start_date field",
			field: "start_date",
			want:  "Ngày bắt đầu",
		},
		{
			name:  "end_date field",
			field: "end_date",
			want:  "Ngày kết thúc",
		},
		{
			name:  "budget_amount field",
			field: "budget_amount",
			want:  "Ngân sách",
		},
		{
			name:  "actual_amount field",
			field: "actual_amount",
			want:  "Chi phí thực tế",
		},
		{
			name:  "unknown field returns original",
			field: "unknown_field",
			want:  "unknown_field",
		},
		{
			name:  "empty field",
			field: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFieldLabel(tt.field)
			if got != tt.want {
				t.Errorf("GetFieldLabel(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}
