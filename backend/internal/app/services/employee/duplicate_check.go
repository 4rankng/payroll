package employee

import (
	"context"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/phone"
)

// DuplicateCheckParams holds the identifiers typed into the create form.
// All optional; at least one must be non-empty.
type DuplicateCheckParams struct {
	CCCD   string
	Mobile string
	Email  string
}

// DuplicateMatch is one existing employee matching the typed identifiers.
// Only identifiers the requester themselves typed are revealed (masked).
type DuplicateMatch struct {
	ID                  uint
	Fullname            string
	CCCDMasked          string  // set only when matched via CCCD
	MobileMasked        string  // set only when matched via mobile
	EmailMasked         string  // set only when matched via email
	CurrentProjectNames []string
	CreatedByName       string
	CreatedAt           time.Time
	MatchedOn           []string // "cccd" | "mobile" | "email"
}

const maxDuplicateMatches = 5

// CheckDuplicates finds existing employees matching the given identifiers.
// Global (unscoped) by design: the point is to catch employees created by other
// partners. Soft-deleted employees are excluded by the repo layer.
func (s *EmployeeService) CheckDuplicates(ctx context.Context, params DuplicateCheckParams) ([]*DuplicateMatch, error) {
	cccd := strings.TrimSpace(params.CCCD)
	mobile := strings.TrimSpace(params.Mobile)
	email := strings.ToLower(strings.TrimSpace(params.Email))

	if cccd == "" && mobile == "" && email == "" {
		return nil, domain.NewValidationError(constants.MsgDuplicateCheckRequiresIdentifierVN)
	}

	// matched[id] -> match accumulator (dedupes the same employee found via
	// multiple identifiers).
	matched := map[uint]*DuplicateMatch{}
	var order []uint

	addMatch := func(e *domain.Employee, field string) {
		m, ok := matched[e.ID]
		if !ok {
			m = &DuplicateMatch{
				ID:            e.ID,
				Fullname:      e.Fullname,
				CreatedByName: e.Creator.Fullname,
				CreatedAt:     e.CreatedAt,
			}
			matched[e.ID] = m
			order = append(order, e.ID)
		}
		switch field {
		case "cccd":
			m.CCCDMasked = maskCCCD(e.CCCD)
		case "mobile":
			m.MobileMasked = maskMobile(e.Mobile)
		case "email":
			m.EmailMasked = maskEmail(derefStringPtr(e.Email))
		}
		m.MatchedOn = append(m.MatchedOn, field)
	}

	if cccd != "" {
		if e, err := s.EmployeeRepo.GetByCCCD(ctx, cccd); err == nil && e != nil {
			addMatch(e, "cccd")
		}
	}

	if mobile != "" {
		// Normalize +84/84/spaced forms to domestic 0XXXXXXXXX; also try the raw
		// value in case stored data is non-normalized.
		candidates := []string{mobile}
		if normalized, err := phone.NormalizeVietnameseMobile(mobile); err == nil && normalized != mobile {
			candidates = append(candidates, normalized)
		}
		for _, candidate := range candidates {
			if e, err := s.EmployeeRepo.GetByMobile(ctx, candidate); err == nil && e != nil {
				addMatch(e, "mobile")
				break
			}
		}
	}

	if email != "" {
		if e, err := s.EmployeeRepo.GetByEmail(ctx, email); err == nil && e != nil {
			addMatch(e, "email")
		}
	}

	if len(order) == 0 {
		return []*DuplicateMatch{}, nil
	}

	result := make([]*DuplicateMatch, 0, len(order))
	for _, id := range order {
		m := matched[id]
		// Active project names for recognition ("this is the person on project X").
		assignments, err := s.ProjectEmployeeRepo.GetByEmployee(ctx, id)
		if err == nil {
			names := make([]string, 0, len(assignments))
			for _, a := range assignments {
				if a.LastDate == nil && a.Project.Name != "" {
					names = append(names, a.Project.Name)
				}
			}
			m.CurrentProjectNames = names
		}
		result = append(result, m)
		if len(result) >= maxDuplicateMatches {
			break
		}
	}

	return result, nil
}

func derefStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// maskCCCD keeps the first 3 and last 4 characters: 031********43
func maskCCCD(cccd string) string {
	return maskKeepEnds(cccd, 3, 4)
}

// maskMobile keeps the last 4 digits: *****3456
func maskMobile(mobile string) string {
	return maskKeepEnds(mobile, 0, 4)
}

// maskEmail keeps the first char of the local part plus the domain: n***@gmail.com
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return maskKeepEnds(email, 1, 0)
	}
	return maskKeepEnds(email[:at], 1, 0) + email[at:]
}

// maskKeepEnds masks the middle of s, keeping the first `keepStart` and last
// `keepEnd` characters. Strings at or below keepStart+keepEnd length are returned as-is.
func maskKeepEnds(s string, keepStart, keepEnd int) string {
	if len(s) <= keepStart+keepEnd {
		return s
	}
	start := ""
	if keepStart > 0 {
		start = s[:keepStart]
	}
	end := ""
	if keepEnd > 0 {
		end = s[len(s)-keepEnd:]
	}
	return start + strings.Repeat("*", len(s)-keepStart-keepEnd) + end
}
