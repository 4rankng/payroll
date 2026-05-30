package converter

// GetFieldLabel returns Vietnamese label for database field names
func GetFieldLabel(field string) string {
	fieldLabels := map[string]string{
		"fullname":            "Tên đầy đủ",
		"email":               "Email",
		"mobile":              "Số điện thoại",
		"cccd":                "CCCD",
		"date_of_birth":       "Ngày sinh",
		"address":             "Địa chỉ",
		"bank_id":             "Ngân hàng",
		"bank_account_name":   "Tên tài khoản",
		"bank_account_number": "Số tài khoản",
		"status":              "Trạng thái",
		"name":                "Tên",
		"code":                "Mã",
		"description":         "Mô tả",
		"start_date":          "Ngày bắt đầu",
		"end_date":            "Ngày kết thúc",
		"budget_amount":       "Ngân sách",
		"actual_amount":       "Chi phí thực tế",
	}

	if label, exists := fieldLabels[field]; exists {
		return label
	}
	return field
}
