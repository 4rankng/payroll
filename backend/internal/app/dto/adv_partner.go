package dto

// UpdateAdvPartnerUserRequest combines employee info + user account updates into one request.
// All fields are optional — only provided fields are updated.
type UpdateAdvPartnerUserRequest struct {
	// Employee fields
	Fullname          *string `json:"fullname,omitempty"`
	Email             *string `json:"email,omitempty" binding:"omitempty,email"`
	CCCD              *string `json:"cccd,omitempty"`
	Mobile            *string `json:"mobile,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
	BankAccountName   *string `json:"bank_account_name,omitempty"`

	// User account fields
	Username *string `json:"username,omitempty" binding:"omitempty,min=3"`
	Password *string `json:"password,omitempty" binding:"omitempty,min=8"`
}
