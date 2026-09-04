package dto

import (
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
	"encoding/json"
	"fmt"
	"time"
)

// CreatePayrateRequest represents the request to create a new payrate
type CreatePayrateRequest struct {
	ProjectID     uint            `json:"-"` // Set from URL path
	Rates         json.RawMessage `json:"rates"`
	EffectiveFrom string          `json:"effective_from" binding:"required"`
	EffectiveTo   *string         `json:"effective_to,omitempty"`
}

// ToDomain converts the DTO to domain entity
func (req *CreatePayrateRequest) ToDomain() (*domain.Payrate, error) {
	// Ensure we have valid JSON rates
	if len(req.Rates) == 0 {
		return nil, fmt.Errorf("rates configuration is required")
	}

	// Validate that it's valid JSON
	var temp map[string]any
	if err := json.Unmarshal(req.Rates, &temp); err != nil {
		return nil, fmt.Errorf("invalid rates JSON format: %w", err)
	}

	return &domain.Payrate{
		ProjectID: req.ProjectID,
		Payrate:   domain.PayrateConfiguration(req.Rates),
	}, nil
}

// UpdatePayrateRequest represents the request to update an existing payrate
type UpdatePayrateRequest struct {
	Rates         json.RawMessage `json:"rates"`
	EffectiveFrom string          `json:"effective_from" binding:"required"`
	EffectiveTo   *string         `json:"effective_to,omitempty"`
}

// ToDomain converts the DTO to domain entity
func (req *UpdatePayrateRequest) ToDomain() (*domain.Payrate, error) {
	// Ensure we have valid JSON rates
	if len(req.Rates) == 0 {
		return nil, fmt.Errorf("rates configuration is required")
	}

	// Validate that it's valid JSON
	var temp map[string]any
	if err := json.Unmarshal(req.Rates, &temp); err != nil {
		return nil, fmt.Errorf("invalid rates JSON format: %w", err)
	}

	return &domain.Payrate{
		Payrate: domain.PayrateConfiguration(req.Rates),
	}, nil
}

// PayrateResponse represents the response containing payrate data
type PayrateResponse struct {
	ID        uint                        `json:"id"`
	ProjectID uint                        `json:"project_id"`
	Rates     domain.PayrateConfiguration `json:"rates"`
	FromDate  string                      `json:"fromDate"`
	ToDate    *string                     `json:"toDate"`
	CreatedBy uint                        `json:"created_by"`
	CreatedAt time.Time                   `json:"created_at"`
	UpdatedAt time.Time                   `json:"updated_at"`
	// EarliestEffectiveFrom is the day after the project's most recent paid
	// timesheet work date — the earliest start date an update may take. Empty
	// when the project has no paid timesheets (no floor beyond today).
	EarliestEffectiveFrom string `json:"earliest_effective_from,omitempty"`
	// FromDateLocked is vestigial: start dates are never load-time locked
	// anymore (later moves split the config via update-as-create). Retained on
	// the wire as always-false for API compatibility; removal is a contract
	// decision. The validate endpoint's per-field locked flag is the live one.
	FromDateLocked bool `json:"from_date_locked"`
}

// ListPayratesRequest represents query parameters for listing payrates
type ListPayratesRequest struct {
	ProjectID *uint  `form:"project_id"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
}

// ListPayratesResponse represents the response for listing payrates with pagination
type ListPayratesResponse struct {
	Success    bool                 `json:"success"`
	Message    string               `json:"message"`
	Data       []PayrateResponse    `json:"data"`
	Pagination *response.Pagination `json:"pagination,omitempty"`
}
