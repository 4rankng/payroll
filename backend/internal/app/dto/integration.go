package dto

// IntegrationOTPRequest is the body for POST /integration/password-reset/otp.
type IntegrationOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// IntegrationVerifyRequest is the body for .../verify.
type IntegrationVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

// IntegrationResetRequest is the body for .../reset. NewPassword is optional;
// when empty the server generates and returns one.
type IntegrationResetRequest struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password"`
}

// IntegrationLookupRequest is the body for POST /integration/employee/lookup.
type IntegrationLookupRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// IntegrationOTPResponse reports the explicit outcome of a send-OTP request.
type IntegrationOTPResponse struct {
	Found             bool    `json:"found"`
	OTPSent           bool    `json:"otp_sent"`
	SessionID         string  `json:"session_id"`
	ExpiresIn         int     `json:"expires_in"`
	OTPLength         int     `json:"otp_length"`
	EmployeeName      string  `json:"employee_name"`
	FailureReason     *string `json:"failure_reason"`
	DeliveryErrorCode int     `json:"delivery_error_code"`
}

// IntegrationVerifyResponse carries the single-use reset token.
type IntegrationVerifyResponse struct {
	Verified   bool   `json:"verified"`
	ResetToken string `json:"reset_token"`
	ExpiresIn  int    `json:"expires_in"`
}

// IntegrationResetResponse carries the login identifier and the new password.
type IntegrationResetResponse struct {
	Username     string `json:"username"`
	NewPassword  string `json:"new_password"`
	EmployeeName string `json:"employee_name"`
}

// IntegrationLookupResponse is the employee detail for chatbot verification.
type IntegrationLookupResponse struct {
	Found        bool   `json:"found"`
	EmployeeName string `json:"employee_name"`
	CCCD         string `json:"cccd"`
	Mobile       string `json:"mobile"`
}
