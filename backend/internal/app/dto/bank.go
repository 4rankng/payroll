package dto

// CreateBankRequest represents the request to create a new bank
type CreateBankRequest struct {
	BranchName string  `json:"branch_name" binding:"required"`
	BankCode   *string `json:"bank_code,omitempty" binding:"omitempty"`
	SwiftCode  *string `json:"swift_code,omitempty" binding:"omitempty"`
	Bin        *string `json:"bin,omitempty" binding:"omitempty,len=6"`
}

// UpdateBankRequest represents the request to update a bank
type UpdateBankRequest struct {
	BranchName *string `json:"branch_name,omitempty"`
	BankCode   *string `json:"bank_code,omitempty"`
	SwiftCode  *string `json:"swift_code,omitempty"`
	Bin        *string `json:"bin,omitempty" binding:"omitempty,len=6"`
	Status     *string `json:"status,omitempty" binding:"omitempty,oneof=active inactive"`
}

// BankResponse represents the response containing bank data
type BankResponse struct {
	ID         uint   `json:"id"`
	BranchName string `json:"branch_name"`
	BankCode   string `json:"bank_code"`
	SwiftCode  string `json:"swift_code"`
	Bin        string `json:"bin"`
}

// ListBanksResponse represents the response for listing banks
type ListBanksResponse struct {
	Banks  []BankResponse `json:"banks"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
