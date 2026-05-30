package utils

import (
	"testing"
)

func TestGenerateUsername(t *testing.T) {
	tests := []struct {
		name     string
		fullname string
		want     string
	}{
		{
			name:     "standard vietnamese name",
			fullname: "Nguyen Thi Hong Tham",
			want:     "thamnth",
		},
		{
			name:     "another vietnamese name",
			fullname: "Dang Quan Khai",
			want:     "khaidq",
		},
		{
			name:     "single name",
			fullname: "Tham",
			want:     "tham",
		},
		{
			name:     "two part name",
			fullname: "Nguyen Tham",
			want:     "thamn",
		},
		{
			name:     "empty string",
			fullname: "",
			want:     "",
		},
		{
			name:     "name with extra spaces",
			fullname: "Nguyen  Thi  Tham",
			want:     "thamnt",
		},
		{
			name:     "lowercase input",
			fullname: "nguyen van a",
			want:     "anv",
		},
		{
			name:     "uppercase input",
			fullname: "NGUYEN VAN B",
			want:     "bnv",
		},
		{
			name:     "name with accents",
			fullname: "Trần Thị Hồng",
			want:     "hongtt",
		},
		{
			name:     "complex vietnamese name",
			fullname: "Lê Hoàng Phúc An",
			want:     "anlhp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUsername(tt.fullname)
			if got != tt.want {
				t.Errorf("GenerateUsername(%q) = %q, want %q", tt.fullname, got, tt.want)
			}
		})
	}
}

func TestEnsureUniqueUsername(t *testing.T) {
	tests := []struct {
		name          string
		baseUsername  string
		existingUsers []string
		want          string
	}{
		{
			name:          "username available",
			baseUsername:  "thamnth",
			existingUsers: []string{},
			want:          "thamnth",
		},
		{
			name:          "username taken once",
			baseUsername:  "thamnth",
			existingUsers: []string{"thamnth"},
			want:          "thamnth02",
		},
		{
			name:          "username taken twice",
			baseUsername:  "thamnth",
			existingUsers: []string{"thamnth", "thamnth02"},
			want:          "thamnth03",
		},
		{
			name:          "username taken multiple times",
			baseUsername:  "khaidq",
			existingUsers: []string{"khaidq", "khaidq02", "khaidq03", "khaidq04"},
			want:          "khaidq05",
		},
		{
			name:          "empty base username",
			baseUsername:  "",
			existingUsers: []string{},
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a checker function that returns true if username exists
			checker := func(username string) bool {
				for _, existing := range tt.existingUsers {
					if username == existing {
						return true
					}
				}
				return false
			}

			got := EnsureUniqueUsername(tt.baseUsername, checker)
			if got != tt.want {
				t.Errorf("EnsureUniqueUsername(%q) = %q, want %q", tt.baseUsername, got, tt.want)
			}
		})
	}
}

func TestEnsureUniqueUsername_AlwaysFalseChecker(t *testing.T) {
	// Test with a checker that always returns false (all usernames available)
	checker := func(username string) bool {
		return false
	}

	result := EnsureUniqueUsername("testuser", checker)
	if result != "testuser" {
		t.Errorf("EnsureUniqueUsername with always-false checker = %q, want %q", result, "testuser")
	}
}

func TestGenerateUsernameEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		fullname string
		wantLen  int
	}{
		{
			name:     "very long name",
			fullname: "Nguyen Thi Hong Tham Anh Mai Phuong",
			wantLen:  12, // "phuong" + 6 initials
		},
		{
			name:     "single character parts",
			fullname: "A B C D",
			wantLen:  4, // "d" + "abc"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUsername(tt.fullname)
			if len(got) != tt.wantLen {
				t.Errorf("GenerateUsername(%q) length = %d, want %d (got: %q)", tt.fullname, len(got), tt.wantLen, got)
			}
		})
	}
}
