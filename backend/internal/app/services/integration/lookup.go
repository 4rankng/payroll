package integration

import (
	"context"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/phone"
)

// LookupResult is the employee detail the chatbot double-checks against what
// the caller reads out. found=false means no single resettable account owns the
// number (absent, or carried by several employee rows).
type LookupResult struct {
	Found        bool
	EmployeeName string
	CCCD         string
	Mobile       string
}

// LookupEmployee resolves a mobile number to the owning account and returns the
// name, CCCD, and mobile for the chatbot to verify. It shares resolveAccount
// with RequestOTP, so lookup and OTP never disagree about whether an account
// exists or what the person's name is. Employee records take precedence over
// the linked user row because employees.mobile/cccd is authoritative for
// employees (the user row may carry a legacy copy or none).
//
// A malformed number is a 400 (validation). Any resolution failure — not found
// or ambiguous — is reported as found=false, never as an error, so the chatbot
// gets a uniform answer.
func (s *Service) LookupEmployee(ctx context.Context, rawPhone string) (*LookupResult, error) {
	if _, err := phone.NormalizeVietnameseMobile(rawPhone); err != nil {
		return nil, domain.NewValidationError(constants.MsgIntegrationPhoneInvalidVN)
	}

	u, emp, err := s.resolveAccount(ctx, rawPhone)
	if err != nil {
		return &LookupResult{Found: false}, nil
	}

	res := &LookupResult{
		Found:        true,
		EmployeeName: displayName(u, emp),
		Mobile:       derefString(u.Mobile),
		CCCD:         derefString(u.CCCD),
	}
	if emp != nil {
		res.Mobile = emp.Mobile
		res.CCCD = emp.CCCD
	}

	return res, nil
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
