package dto

// UpdateProfileRequest represents the request to update the current user's profile
type UpdateProfileRequest struct {
	Email    *string `json:"email"`
	Fullname *string `json:"fullname"`
	Password *string `json:"password" binding:"omitempty,min=8"`
	CCCD     *string `json:"cccd"`
	Mobile   *string `json:"mobile"`
}

// GoogleLoginRequest represents the request to login with Google OAuth ID token
type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}
