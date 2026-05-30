package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBank_ValidateBranchName(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "Valid branch name",
			branchName: "Saigon Branch",
			wantErr:    false,
		},
		{
			name:       "Empty branch name",
			branchName: "",
			wantErr:    true,
			errMsg:     "branch name is required",
		},
		{
			name:       "Branch name too long (over 255 chars)",
			branchName: string(make([]byte, 256)),
			wantErr:    true,
			errMsg:     "branch name must be less than 255 characters",
		},
		{
			name:       "Valid long branch name (255 chars)",
			branchName: string(make([]byte, 255)),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bank := &Bank{
				BranchName: tt.branchName,
			}
			err := bank.ValidateBranchName()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBank_IsValid(t *testing.T) {
	tests := []struct {
		name       string
		bank       Bank
		wantErr    bool
		errContain string
	}{
		{
			name: "Valid bank",
			bank: Bank{
				BranchName: "Saigon Branch",
			},
			wantErr: false,
		},
		{
			name: "Invalid - empty branch name",
			bank: Bank{
				BranchName: "",
			},
			wantErr:    true,
			errContain: "branch name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.bank.IsValid()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBank_CanBeUsed(t *testing.T) {
	tests := []struct {
		name     string
		bank     Bank
		expected bool
	}{
		{
			name: "Can be used - valid bank",
			bank: Bank{
				BranchName: "Saigon Branch",
			},
			expected: true,
		},
		{
			name: "Cannot be used - empty branch name",
			bank: Bank{
				BranchName: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bank.CanBeUsed()
			assert.Equal(t, tt.expected, result)
		})
	}
}
