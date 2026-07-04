package dto

import (
	"api-server/internal/domain"
	"time"
)

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=8"`
	Fullname string `json:"fullname" binding:"required"`
	Role     string `json:"role,omitempty"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email    *string `json:"email"`
	Username *string `json:"username" binding:"omitempty,min=3"`
	Fullname *string `json:"fullname"`
	Role     *string `json:"role"`
}

// UserResponse represents the response containing user data
type UserResponse struct {
	ID        uint       `json:"id"`
	Email     string     `json:"email"`
	Username  string     `json:"username"`
	Fullname  string     `json:"fullname"`
	Role      string     `json:"role"`
	CCCD      string     `json:"cccd,omitempty"`
	Mobile    string     `json:"mobile,omitempty"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=1"`
	Password string `json:"password" binding:"required"`
}

// VerifyOTPRequest completes the email-OTP login step. OTPSessionID was returned
// by /auth/login when the second factor was required; Code is the 6-digit value
// emailed to the user.
type VerifyOTPRequest struct {
	OTPSessionID string `json:"otp_session_id" binding:"required"`
	Code         string `json:"code" binding:"required"`
}

// ResendOTPRequest re-issues an OTP code for a pending session (email didn't
// arrive). The same session id is returned; the prior code becomes invalid.
type ResendOTPRequest struct {
	OTPSessionID string `json:"otp_session_id" binding:"required"`
}

// LoginResponse represents the login response. For a normal login, User +
// AccessToken are populated. When the email-OTP second factor is required
// (admin/partner, OTP_ENABLE=true), OTPRequired + OTPSessionID are populated
// instead and AccessToken is empty — the client must POST the code to
// /auth/login/verify to obtain the real token.
type LoginResponse struct {
	User        *UserResponse `json:"user,omitempty"`
	AccessToken string        `json:"access_token,omitempty"`
	TokenType   string        `json:"token_type,omitempty"`
	ExpiresIn   int64         `json:"expires_in,omitempty"`
	// OTP-step fields (present only when the second factor is required):
	OTPRequired  bool   `json:"otp_required,omitempty"`
	OTPSessionID string `json:"otp_session_id,omitempty"`
}

// ListUsersRequest represents the request to list users with pagination
type ListUsersRequest struct {
	Page           int    `form:"page,default=1" binding:"min=1"`
	PageSize       int    `form:"pageSize,default=100" binding:"min=1,max=500"`
	SortBy         string `form:"sortBy,default=created_at"`
	SortOrder      string `form:"sortOrder,default=desc" binding:"omitempty,oneof=asc desc"`
	Role           string `form:"role" binding:"omitempty,oneof=admin partner employee"`
	Search         string `form:"search"`
	IDs            string `form:"ids"` // Comma-separated list of IDs (e.g., "1,2,3")
	LastLoginToday bool   `form:"last_login_today"`
}

// ToUserResponse converts a domain User to UserResponse
func ToUserResponse(user *domain.User) UserResponse {
	var email string
	if user.Email != nil {
		email = *user.Email
	}

	var cccd string
	if user.CCCD != nil {
		cccd = *user.CCCD
	}

	var mobile string
	if user.Mobile != nil {
		mobile = *user.Mobile
	}

	return UserResponse{
		ID:        user.ID,
		Email:     email,
		Username:  user.Username,
		Fullname:  user.Fullname,
		Role:      string(user.Role),
		CCCD:      cccd,
		Mobile:    mobile,
		LastLogin: user.LastLogin,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// ToUserResponseList converts a slice of domain Users to UserResponse slice
func ToUserResponseList(users []*domain.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = ToUserResponse(user)
	}
	return responses
}

// Password related DTOs
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type PasswordStrengthRequest struct {
	Password string `json:"password" binding:"required"`
}

type PasswordStrengthResponse struct {
	Score        int      `json:"score"`                   // 0-100
	Strength     string   `json:"strength"`                // Vietnamese strength label
	Feedback     []string `json:"feedback"`                // Improvement suggestions
	IsValid      bool     `json:"is_valid"`                // Whether password meets requirements
	ErrorMessage string   `json:"error_message,omitempty"` // Validation error if any
}

// UserSummaryResponse represents the user statistics summary
type UserSummaryResponse struct {
	TotalUsers        int64 `json:"total_users"`
	TotalAdmins       int64 `json:"total_admins"`
	TotalPartners     int64 `json:"total_partners"`
	TotalEmployees    int64 `json:"total_employees"`
	RecentLoginsToday int64 `json:"recent_logins_today"`
}

// ResetUserPasswordRequest represents the request to reset a user's password
type ResetUserPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

// UserActivitiesRequest represents the request parameters for user activities
type UserActivitiesRequest struct {
	Days int `form:"days,default=30" binding:"min=1,max=365"`
}

// UserActivitiesResponse represents the user activity summary with essential metrics
type UserActivitiesResponse struct {
	UserID            uint                         `json:"user_id"`
	PeriodDays        int                          `json:"period_days"`
	Authentication    AuthenticationMetrics        `json:"authentication"`
	PayrollOperations PayrollOperationMetrics      `json:"payroll_operations"`
	RecentActivities  []UserActivityDetailResponse `json:"recent_activities"`
}

// AuthenticationMetrics represents authentication-related metrics
type AuthenticationMetrics struct {
	TotalLogins int64      `json:"total_logins"`
	LastLogin   *time.Time `json:"last_login"`
}

// PayrollOperationMetrics represents payroll-specific operation metrics
type PayrollOperationMetrics struct {
	TimesheetsManaged int64 `json:"timesheets_managed"`
}

// UserActivityDetailResponse represents a detailed activity entry
type UserActivityDetailResponse struct {
	ID        uint      `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// ResetFirstTimeLoginPasswordResponse represents the response for bulk reset first-time login passwords
type ResetFirstTimeLoginPasswordResponse struct {
	Total     int      `json:"Total"`
	Usernames []string `json:"Usernames"`
}

// PasswordResetJobStatusResponse represents the status of the password reset job
type PasswordResetJobStatusResponse struct {
	Status    JobStatus `json:"status"`
	Total     *int      `json:"total,omitempty"`
	Usernames []string  `json:"usernames,omitempty"`
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
)

// StartPasswordResetJobResponse represents the response when starting a job
type StartPasswordResetJobResponse struct {
	Status JobStatus `json:"status"`
}
