package workers

import (
	"fmt"
	"math/big"
	"sort"

	"api-server/internal/domain"
)

// allocatePaidTransferAmount apportions the persisted transfer by gross wage.
// Integer remainders go to the largest fractional shares (ID breaks ties), so
// the recorded sum always equals the bank transfer without floating-point loss.
func allocatePaidTransferAmount(timesheets []*domain.Timesheet, amount int64) (map[uint]int64, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}
	total := new(big.Int)
	seen := make(map[uint]bool, len(timesheets))
	for _, timesheet := range timesheets {
		if timesheet == nil || timesheet.Amount < 0 {
			return nil, fmt.Errorf("invalid timesheet wage")
		}
		if seen[timesheet.ID] {
			return nil, fmt.Errorf("duplicate timesheet %d", timesheet.ID)
		}
		seen[timesheet.ID] = true
		total.Add(total, big.NewInt(timesheet.Amount))
	}
	if total.Sign() == 0 {
		return nil, fmt.Errorf("transfer has no positive timesheet wages")
	}
	type fractionalShare struct {
		id        uint
		remainder *big.Int
	}
	shares := make([]fractionalShare, 0, len(timesheets))
	result := make(map[uint]int64, len(timesheets))
	remaining := amount
	for _, timesheet := range timesheets {
		numerator := new(big.Int).Mul(big.NewInt(timesheet.Amount), big.NewInt(amount))
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(numerator, total, remainder)
		result[timesheet.ID] = quotient.Int64()
		remaining -= quotient.Int64()
		shares = append(shares, fractionalShare{timesheet.ID, remainder})
	}
	sort.Slice(shares, func(i, j int) bool {
		if comparison := shares[i].remainder.Cmp(shares[j].remainder); comparison != 0 {
			return comparison > 0
		}
		return shares[i].id < shares[j].id
	})
	for i := int64(0); i < remaining; i++ {
		result[shares[i].id]++
	}
	return result, nil
}
