package domain

import (
	"encoding/json"
	"time"
)

// AssetImportMetadata contains import progress tracking data for flex_pay_import assets
type AssetImportMetadata struct {
	TotalRows     *int       `json:"total_rows,omitempty"`
	ProcessedRows *int       `json:"processed_rows,omitempty"`
	Status        *string    `json:"status,omitempty"`
	ForMonth      *string    `json:"for_month,omitempty"`
	Result        *string    `json:"result,omitempty"`
	Error         *string    `json:"error,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// IsCompleted returns true if the import is completed
func (m *AssetImportMetadata) IsCompleted() bool {
	return m != nil && m.Status != nil && *m.Status == "completed"
}

// IsProcessing returns true if the import is in progress
func (m *AssetImportMetadata) IsProcessing() bool {
	return m != nil && m.Status != nil && *m.Status == "processing"
}

// IsFailed returns true if the import failed
func (m *AssetImportMetadata) IsFailed() bool {
	return m != nil && m.Status != nil && *m.Status == "failed"
}

// GetPercentage calculates the completion percentage
func (m *AssetImportMetadata) GetPercentage() int {
	if m == nil || m.TotalRows == nil || *m.TotalRows == 0 {
		return 0
	}
	percentage := float64(*m.ProcessedRows) / float64(*m.TotalRows) * 100
	return int(percentage)
}

// NewAssetImportMetadata creates a new import metadata with default values
func NewAssetImportMetadata() *AssetImportMetadata {
	status := "pending"
	return &AssetImportMetadata{
		Status: &status,
	}
}

// ToJSON converts the metadata to JSON string
func (m *AssetImportMetadata) ToJSON() (*string, error) {
	if m == nil {
		return nil, nil
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	jsonStr := string(jsonBytes)
	return &jsonStr, nil
}

// FromJSON creates AssetImportMetadata from JSON string
func FromJSON(jsonStr *string) (*AssetImportMetadata, error) {
	if jsonStr == nil || *jsonStr == "" {
		return NewAssetImportMetadata(), nil
	}

	var metadata AssetImportMetadata
	err := json.Unmarshal([]byte(*jsonStr), &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}
