package timesheet

import (
	"testing"
)

func TestValidator_ParsePaytype(t *testing.T) {
	v := &Validator{} // No dependencies needed for ParsePaytype

	tests := []struct {
		name         string
		paytype      string
		wantPosition string
		wantDayType  string
		wantHourType string
	}{
		{
			name:         "standard three-part format",
			paytype:      "kỹ sư.ngày thường.08:00-17:00",
			wantPosition: "kỹ sư",
			wantDayType:  "ngày thường",
			wantHourType: "08:00-17:00",
		},
		{
			name:         "three-part with overtime",
			paytype:      "phổ thông.ngày lễ.tăng ca",
			wantPosition: "phổ thông",
			wantDayType:  "ngày lễ",
			wantHourType: "tăng ca",
		},
		{
			name:         "two-part legacy format",
			paytype:      "ngày thường.08:00-17:00",
			wantPosition: "phổ thông",
			wantDayType:  "ngày thường",
			wantHourType: "08:00-17:00",
		},
		{
			name:         "single part format",
			paytype:      "tăng ca",
			wantPosition: "phổ thông",
			wantDayType:  "Ngày thường",
			wantHourType: "tăng ca",
		},
		{
			name:         "complex hour type with dots",
			paytype:      "kỹ sư.ngày nghỉ.17:00-22:00",
			wantPosition: "kỹ sư",
			wantDayType:  "ngày nghỉ",
			wantHourType: "17:00-22:00",
		},
		{
			name:         "empty string",
			paytype:      "",
			wantPosition: "phổ thông",
			wantDayType:  "Ngày thường",
			wantHourType: "",
		},
		{
			name:         "position with special characters",
			paytype:      "trưởng phòng.ngày thường.08:00-17:00",
			wantPosition: "trưởng phòng",
			wantDayType:  "ngày thường",
			wantHourType: "08:00-17:00",
		},
		{
			name:         "multiple dots in hour type",
			paytype:      "kỹ sư.ngày lễ.08:00-12:00.13:00-17:00",
			wantPosition: "kỹ sư",
			wantDayType:  "ngày lễ",
			wantHourType: "08:00-12:00.13:00-17:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPosition, gotDayType, gotHourType := v.ParsePaytype(tt.paytype)
			if gotPosition != tt.wantPosition {
				t.Errorf("ParsePaytype() position = %v, want %v", gotPosition, tt.wantPosition)
			}
			if gotDayType != tt.wantDayType {
				t.Errorf("ParsePaytype() dayType = %v, want %v", gotDayType, tt.wantDayType)
			}
			if gotHourType != tt.wantHourType {
				t.Errorf("ParsePaytype() hourType = %v, want %v", gotHourType, tt.wantHourType)
			}
		})
	}
}
