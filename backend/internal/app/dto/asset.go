package dto

import "time"

type AssetUploadResponse struct {
	ID         uint      `json:"id"`
	Filename   string    `json:"filename"`
	UploadType string    `json:"upload_type"`
	UploadedBy uint      `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type AssetResponse struct {
	ID         uint      `json:"id"`
	Filename   string    `json:"filename"`
	UploadType string    `json:"upload_type"`
	UploadedBy uint      `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type AssetListRequest struct {
	UploadType string `form:"upload_type"`
	UploadedBy uint   `form:"uploaded_by"`
	Limit      int    `form:"limit"`
	Offset     int    `form:"offset"`
	SortBy     string `form:"sort_by"`
	SortOrder  string `form:"sort_order"`
}

type AssetUploadForm struct {
	UploadType string `form:"upload_type" binding:"required"`
}
