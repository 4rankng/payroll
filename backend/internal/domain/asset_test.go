package domain

import "testing"

// The day-9 roster job stores its workbook with UploadTypeCheckInRoster. The
// constant must stay in allowedUploadTypes: the map is a closed allowlist (path
// traversal guard), and a constant missing from it makes UploadAssetFromBytes
// reject every upload — the job would log a failure every month and the
// dashboard banner would never appear.
func TestIsValidUploadTypeAcceptsCheckInRoster(t *testing.T) {
	if !IsValidUploadType(UploadTypeCheckInRoster) {
		t.Fatalf("UploadTypeCheckInRoster (%q) must be an allowed upload type", UploadTypeCheckInRoster)
	}
}

func TestIsValidUploadTypeRejectsUnknown(t *testing.T) {
	for _, uploadType := range []string{"", "../etc", "check_in_roster", "CHECKIN_ROSTER"} {
		if IsValidUploadType(uploadType) {
			t.Fatalf("upload type %q must be rejected", uploadType)
		}
	}
}
