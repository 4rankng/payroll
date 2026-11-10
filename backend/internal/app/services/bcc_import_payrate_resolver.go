package services

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/domain"
)

// resolvedConfig is one payrate configuration with its rates flattened once,
// so per-entry-date lookups never re-parse JSON.
type resolvedConfig struct {
	id        uint
	fromDate  time.Time
	flatRates map[string]int
}

// bccPositionAvailability records when a position becomes priceable inside an
// import month. availableFrom is the first day of the month (or the config's
// own start date, when it begins mid-month) on which the position can price a
// row — the "có từ dd/mm" hint of the unified missing-position error.
type bccPositionAvailability struct {
	name          string
	availableFrom time.Time
}

// payrateResolver is the single authority for payrate resolution in every BCC
// import path (PAYR-36). Business rule: the config for a work date is the one
// with the NEAREST from_date at or before that date — the same
// MAX(from_date) WHERE from_date <= date contract as
// PayrateRepository.GetActiveByProjectAndDate, ties broken by the higher id.
// No import path may anchor one config for a whole file or validate a whole
// sheet against one config:
//
//   - positions validate against the UNION of positions across the configs
//     effective at some day of the import month (monthPositions), and
//   - every rate lookup resolves against the config of its own work date
//     (ratesFor / rateToTargetFor).
type payrateResolver struct {
	// configs is sorted by (fromDate asc, id asc); ascending iteration makes
	// "last hit" the nearest from_date and, on a same-date tie, the higher id.
	configs []resolvedConfig
	// rateToTargetCache memoizes buildRateToTarget per (date, position prefix);
	// imports are single-goroutine so no locking is needed.
	rateToTargetCache map[string]map[int]rateTarget
}

// newPayrateResolver flattens every config once and orders the timeline.
// loc must match the import month's location (time.Local — see
// prepareImportContext) so date comparisons behave like the SQL comparisons
// of GetActiveByProjectAndDate.
func newPayrateResolver(configs []*domain.Payrate, loc *time.Location) (*payrateResolver, error) {
	r := &payrateResolver{rateToTargetCache: make(map[string]map[int]rateTarget)}
	for _, p := range configs {
		flatRates, err := p.Payrate.Flatten()
		if err != nil {
			return nil, fmt.Errorf("config id %d: %v", p.ID, err)
		}
		// Normalize to midnight in loc: from_date is a DATE column and the
		// driver scans it at midnight in the connection location; rebuild in
		// loc defensively so comparisons never see stray hours.
		from := time.Date(p.FromDate.In(loc).Year(), p.FromDate.In(loc).Month(), p.FromDate.In(loc).Day(), 0, 0, 0, 0, loc)
		r.configs = append(r.configs, resolvedConfig{id: p.ID, fromDate: from, flatRates: flatRates})
	}
	sort.SliceStable(r.configs, func(i, j int) bool {
		if !r.configs[i].fromDate.Equal(r.configs[j].fromDate) {
			return r.configs[i].fromDate.Before(r.configs[j].fromDate)
		}
		return r.configs[i].id < r.configs[j].id
	})
	return r, nil
}

// configFor returns the payrate config active on date: the nearest from_date
// at or before it (MAX(from_date) WHERE from_date <= date, higher id on ties).
func (r *payrateResolver) configFor(date time.Time) (resolvedConfig, bool) {
	var found *resolvedConfig
	for i := range r.configs {
		c := &r.configs[i]
		if !c.fromDate.After(date) {
			found = c
		}
	}
	if found == nil {
		return resolvedConfig{}, false
	}
	return *found, true
}

// ratesFor returns the flattened rates of the config active on date, or nil
// when no config covers it (the weekly paths treat nil/empty as "no config
// for this date").
func (r *payrateResolver) ratesFor(date time.Time) map[string]int {
	c, ok := r.configFor(date)
	if !ok {
		return nil
	}
	return c.flatRates
}

// rateToTargetFor returns the (cached) rate→bucket map for one work date,
// scoped to posPrefix ("" = all positions, legacy single-position behavior).
func (r *payrateResolver) rateToTargetFor(date time.Time, posPrefix string) map[int]rateTarget {
	key := date.Format("2006-01-02") + "\x00" + posPrefix
	if cached, ok := r.rateToTargetCache[key]; ok {
		return cached
	}
	built := buildRateToTarget(r.ratesFor(date), posPrefix)
	r.rateToTargetCache[key] = built
	return built
}

// coversMonth reports whether at least one config is active on some day of
// [monthStart, monthEnd]. monthEnd always lands on the nearest config when
// one starts at or before it, so probing monthEnd is exact.
func (r *payrateResolver) coversMonth(monthStart, monthEnd time.Time) bool {
	_, ok := r.configFor(monthEnd)
	return ok
}

// monthPositions unions the positions of every config effective at some day
// of the import month (PAYR-36 business rule). A config is effective inside
// the month when its from_date is at or before monthEnd and it is not
// superseded before monthStart; each of its positions is available from
// max(from_date, monthStart). The earliest config carrying a position wins
// its display spelling and availableFrom, so a position added mid-month
// reports the day it actually became priceable.
func (r *payrateResolver) monthPositions(monthStart, monthEnd time.Time) map[string]bccPositionAvailability {
	avail := make(map[string]bccPositionAvailability)
	for i := range r.configs {
		c := &r.configs[i]
		if c.fromDate.After(monthEnd) {
			break // ascending order — nothing later can start inside the month
		}
		if i+1 < len(r.configs) && !r.configs[i+1].fromDate.After(monthStart) {
			continue // superseded before the month began: never active inside it
		}
		from := c.fromDate
		if from.Before(monthStart) {
			from = monthStart
		}
		for path := range c.flatRates {
			position, _, _ := strings.Cut(path, ".")
			if position == "" {
				continue
			}
			key := bccNormName(position)
			if existing, ok := avail[key]; ok {
				if from.Before(existing.availableFrom) {
					existing.availableFrom = from
					avail[key] = existing
				}
				continue
			}
			avail[key] = bccPositionAvailability{name: position, availableFrom: from}
		}
	}
	return avail
}

// positionNamesForMonth returns the sorted display names of monthPositions —
// the position vocabulary every import path validates against.
func (r *payrateResolver) positionNamesForMonth(monthStart, monthEnd time.Time) []string {
	avail := r.monthPositions(monthStart, monthEnd)
	names := make([]string, 0, len(avail))
	for _, a := range avail {
		names = append(names, a.name)
	}
	sort.Strings(names)
	return names
}

// missingPositionMessage builds the unified actionable error for a position
// absent from every config effective in the import month (PAYR-36). When the
// position exists in a config outside the month the message says where:
// a future config hints the start date, a superseded past config says the
// position existed but no longer applies. Always lists the month's available
// positions so the uploader can align the sheet names.
func (r *payrateResolver) missingPositionMessage(position string, monthStart, monthEnd time.Time) string {
	hint := ""
	for i := range r.configs {
		c := &r.configs[i]
		if !resolvedConfigHasPosition(c, position) {
			continue
		}
		if c.fromDate.After(monthEnd) {
			hint = fmt.Sprintf(" (sẽ có từ %s)", c.fromDate.Format("02/01"))
		} else {
			hint = fmt.Sprintf(" (đã có từ %s nhưng không còn hiệu lực trong tháng %s)",
				c.fromDate.Format("02/01"), monthStart.Format("01/2006"))
		}
		break
	}
	available := r.positionNamesForMonth(monthStart, monthEnd)
	if len(available) == 0 {
		available = []string{"(không có vị trí nào)"}
	}
	return fmt.Sprintf("Vị trí %q chưa có trong bảng lương của dự án%s — cần cập nhật bảng lương hiệu lực từ đầu kỳ hoặc import theo kỳ config phù hợp. Các vị trí khả dụng trong tháng %s: %s",
		position, hint, monthStart.Format("01/2006"), strings.Join(available, ", "))
}

// resolvedConfigHasPosition reports whether the config's flattened paths
// carry the position (normalized comparison, same rule as sheet matching).
func resolvedConfigHasPosition(c *resolvedConfig, position string) bool {
	want := bccNormName(position)
	for path := range c.flatRates {
		first, _, _ := strings.Cut(path, ".")
		if bccNormName(first) == want {
			return true
		}
	}
	return false
}
