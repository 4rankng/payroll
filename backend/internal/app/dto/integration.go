package dto

import (
	"time"

	"api-server/internal/domain"
)

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

// ZaloTokenResponse is the current OA access token for the TingTing chatbot's
// pull lane (GET /integration/zalo/token). Payroll is the sole rotator of the
// shared OA token pair; the refresh token never leaves payroll.
type ZaloTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// --- Self check-in via chatbot (TingTing OA) ---------------------------------

// SelfCheckinOTPRequest is the body for POST /integration/self-checkin/otp.
type SelfCheckinOTPRequest struct {
	Phone string `json:"phone" binding:"required"`
}

// SelfCheckinVerifyRequest is the body for .../verify.
type SelfCheckinVerifyRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

// SelfCheckinUpdateRequest is the body for .../update. Enable is a pointer so
// an explicit false is distinguished from a missing field.
type SelfCheckinUpdateRequest struct {
	ActionToken string `json:"action_token" binding:"required"`
	ProjectID   uint   `json:"project_id" binding:"required"`
	Enable      *bool  `json:"enable" binding:"required"`
}

// SelfCheckinPendingChangeDTO reports one queued check-in change on an
// assignment. Type is "enable" or "disable"; effective_from is day 1 of the
// month the change lands.
type SelfCheckinPendingChangeDTO struct {
	Type          string    `json:"type"`
	EffectiveFrom time.Time `json:"effective_from"`
}

// SelfCheckinTargetDTO carries the project's attendance geofence.
type SelfCheckinTargetDTO struct {
	RadiusMeters uint                  `json:"radius_meters"`
	Gates        []domain.GeofenceGate `json:"gates"`
}

// SelfCheckinShiftWindowDTO is one advisory shift and its permitted
// check-in/check-out periods, mirroring the profile endpoint's windows.
// Values are absolute instants in the application timezone (Asia/Ho_Chi_Minh),
// serialized as RFC 3339.
type SelfCheckinShiftWindowDTO struct {
	ShiftStart          time.Time `json:"shift_start"`
	ShiftEnd            time.Time `json:"shift_end"`
	CheckInWindowStart  time.Time `json:"check_in_window_start"`
	CheckInWindowEnd    time.Time `json:"check_in_window_end"`
	CheckOutWindowStart time.Time `json:"check_out_window_start"`
	CheckOutWindowEnd   time.Time `json:"check_out_window_end"`
	// ShiftName is the admin-chosen display name for this shift's time-range
	// (e.g. "Ca ngày"). Empty when no name is configured.
	ShiftName string `json:"shift_name,omitempty"`
}

// SelfCheckinAssignmentDTO is one active assignment on a supported project.
type SelfCheckinAssignmentDTO struct {
	ProjectID        uint                         `json:"project_id"`
	ProjectName      string                       `json:"project_name"`
	CheckInEnabled   bool                         `json:"check_in_enabled"`
	PendingChange    *SelfCheckinPendingChangeDTO `json:"pending_change,omitempty"`
	CheckInStartDate *time.Time                   `json:"check_in_start_date,omitempty"`
	PaymentSchedule  string                       `json:"payment_schedule"`
	CheckInTarget    *SelfCheckinTargetDTO        `json:"check_in_target,omitempty"`
	ShiftWindows     []SelfCheckinShiftWindowDTO  `json:"shift_windows"`
}

// SelfCheckinStatusRequest carries the verified action token for a read-only
// state query. The token is peeked, never consumed.
type SelfCheckinStatusRequest struct {
	ActionToken string `json:"action_token" binding:"required"`
}

// SelfCheckinStatusResponse reports the employee's current self check-in
// state: one entry per supported project with its on/off flag, start date and
// any queued change, so the bot answers state questions without mutating.
type SelfCheckinStatusResponse struct {
	Found        bool                       `json:"found"`
	EmployeeName string                     `json:"employee_name"`
	Assignments  []SelfCheckinAssignmentDTO `json:"assignments"`
}

// SelfCheckinOTPResponse reports the explicit outcome of a self check-in OTP
// request. On success assignments carries the eligible projects so the bot
// needs one round-trip before asking the employee to verify.
type SelfCheckinOTPResponse struct {
	Found             bool                       `json:"found"`
	OTPSent           bool                       `json:"otp_sent"`
	SessionID         string                     `json:"session_id"`
	ExpiresIn         int                        `json:"expires_in"`
	OTPLength         int                        `json:"otp_length"`
	EmployeeName      string                     `json:"employee_name"`
	FailureReason     *string                    `json:"failure_reason"`
	DeliveryErrorCode int                        `json:"delivery_error_code"`
	Assignments       []SelfCheckinAssignmentDTO `json:"assignments"`
}

// SelfCheckinVerifyResponse carries the single-use action token.
type SelfCheckinVerifyResponse struct {
	Verified    bool   `json:"verified"`
	ActionToken string `json:"action_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// SelfCheckinUpdateResponse is the enable/disable verdict, shaped so the bot
// can phrase day-1-8 (immediate) vs day-9+ (next month) wording without extra
// calls. effective_from is omitted when a queued enable was merely cancelled.
type SelfCheckinUpdateResponse struct {
	Success                bool       `json:"success"`
	Kind                   string     `json:"kind"`
	Immediate              bool       `json:"immediate"`
	EffectiveFrom          *time.Time `json:"effective_from,omitempty"`
	CancelledPendingEnable bool       `json:"cancelled_pending_enable"`
	// CancelledPendingDisable is true when the enable superseded a queued
	// disable: the service was never off (it stays on from the original start
	// date), so the bot confirms the cancellation instead of announcing a
	// fresh activation.
	CancelledPendingDisable bool `json:"cancelled_pending_disable"`
}
