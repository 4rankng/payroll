package bank

import (
	"sort"
	"strings"
)

// Keyword represents a keyword and its corresponding bank name
// The bank name must match exact branch_name value in the banks table
type Keyword struct {
	Keyword string
	Bank    string // Must match exact branch_name in database
}

// keywords contains all keyword mappings for bank name resolution.
// Order does NOT matter — keywords are auto-sorted by length at init time.
var keywords = []Keyword{
	// Vietcombank
	{"VCB", "Ngoại thương Việt Nam (VCB)"},
	{"VIETCOMBANK", "Ngoại thương Việt Nam (VCB)"},
	{"NGOAITHUONG", "Ngoại thương Việt Nam (VCB)"},

	// Techcombank (Kỹ Thương)
	{"TCB", "Kỹ Thương (TCB)"},
	{"TECHCOMBANK", "Kỹ Thương (TCB)"},
	{"KYTHUONG", "Kỹ Thương (TCB)"},

	// Agribank (Nông nghiệp)
	{"VBA", "Nông nghiệp và Phát triển nông thôn (VBA)"},
	{"AGRIBANK", "Nông nghiệp và Phát triển nông thôn (VBA)"},
	{"NONGNGHIEP", "Nông nghiệp và Phát triển nông thôn (VBA)"},

	// MB (Quân đội)
	{"MB", "Quân đội (MB)"},
	{"MBBANK", "Quân đội (MB)"},
	{"QUANDOI", "Quân đội (MB)"},

	// MSB (Hàng hải)
	{"MSB", "Hàng hải (MSB)"},
	{"MSBBANK", "Hàng hải (MSB)"},
	{"HANGHAI", "Hàng hải (MSB)"},

	// ABBank (An Bình)
	{"ABBANK", "An Bình (ABBANK)"},
	{"AB", "An Bình (ABBANK)"},
	{"ANBINH", "An Bình (ABBANK)"},

	// BIDV (Đầu tư và Phát triển)
	{"BIDV", "Đầu tư và Phát triển (BIDV)"},
	{"DAUTU", "Đầu tư và Phát triển (BIDV)"},

	// VietinBank (Công Thương Việt Nam)
	{"VIETINBANK", "Công Thương Việt Nam (VIETINBANK)"},
	{"VIETIN", "Công Thương Việt Nam (VIETINBANK)"},
	{"CONGTHUONG", "Công Thương Việt Nam (VIETINBANK)"},

	// VPBank (Việt Nam Thịnh Vượng)
	{"VPBANK", "Việt Nam Thịnh Vượng (VPB)"},
	{"VPB", "Việt Nam Thịnh Vượng (VPB)"},
	{"THINHVUONG", "Việt Nam Thịnh Vượng (VPB)"},

	// Sacombank
	{"SACOMBANK", "Sacombank (STB)"},
	{"SGB", "Sài Gòn Công thương (SGB)"},
	{"SAIGONCONGTHUONG", "Sài Gòn Công thương (SGB)"},

	// SHB (Sài Gòn Hà Nội)
	{"SHB", "Sài Gòn Hà Nội (SHB)"},
	{"SAIGONHANOI", "Sài Gòn Hà Nội (SHB)"},

	// SCB (Sài Gòn)
	{"SCB", "Sài Gòn (SCB)"},

	// VIB (Quốc tế)
	{"VIB", "Quốc tế (VIB)"},
	{"QUOCTE", "Quốc tế (VIB)"},
	{"VIETCAPITAL", "Quốc tế (VIB)"},

	// OCB (Phương Đông)
	{"OCB", "Phương Đông (OCB)"},
	{"PHUONGDONG", "Phương Đông (OCB)"},
	{"ORIENTAL", "Phương Đông (OCB)"},

	// ACB (Á Châu)
	{"ACB", "Á Châu (ACB)"},
	{"ASIA", "Á Châu (ACB)"},
	{"ACHAU", "Á Châu (ACB)"},

	// TPBank (Tiên Phong)
	{"TPB", "Tiên Phong (TPB)"},
	{"TPBANK", "Tiên Phong (TPB)"},
	{"TIENPHONG", "Tiên Phong (TPB)"},

	// HDBank (Phát triển nhà TP HCM)
	{"HDB", "Phát triển nhà TP HCM (HDB)"},
	{"HDBANK", "Phát triển nhà TP HCM (HDB)"},

	// Eximbank (Xuất Nhập Khẩu)
	{"EIB", "Xuất Nhập Khẩu (EIB)"},
	{"EXIMBANK", "Xuất Nhập Khẩu (EIB)"},
	{"XUATNK", "Xuất Nhập Khẩu (EIB)"},

	// Kienlongbank (Kiên Long)
	{"KLB", "Kiên Long (KLB)"},
	{"KIENLONGBANK", "Kiên Long (KLB)"},

	// Nam A Bank
	{"NAMABANK", "Nam Á (NAMABANK)"},
	{"NAMA", "Nam Á (NAMABANK)"},

	// DongA Bank (Đông Á)
	{"DAB", "Đông Á (DAB)"},
	{"DONGABANK", "Đông Á (DAB)"},
	{"DONGA", "Đông Á (DAB)"},

	// PG Bank (Ngân hàng TMCP Thượng vượng và Phát triển)
	{"PGB", "Ngân hàng TMCP Thịnh vượng và Phát triển (PGB)"},
	{"PGBANK", "Ngân hàng TMCP Thịnh vượng và Phát triển (PGB)"},

	// PVcomBank (Đại chúng Việt Nam) — formerly OceanBank/PVC
	{"PVC", "Đại chúng Việt Nam (PVC)"},
	{"PVCOMBANK", "Đại chúng Việt Nam (PVC)"},
	{"OCEANBANK", "Đại chúng Việt Nam (PVC)"},
	{"DAICHUNG", "Đại chúng Việt Nam (PVC)"},

	// VRB (Liên Doanh Việt Nga)
	{"VRB", "Liên Doanh Việt Nga (VRB)"},

	// Woori Bank
	{"WOORI", "Woori Việt Nam (Woori)"},

	// UOB
	{"UOB", "United Overseas Bank Việt Nam (UOB)"},

	// Indovina
	{"IVB", "Indovina (IVB)"},
	{"INDOVINA", "Indovina (IVB)"},

	// Shinhan (MTV Shinhan Việt Nam)
	{"SHBVN", "MTV Shinhan Việt Nam (SHBVN)"},
	{"SHINHAN", "MTV Shinhan Việt Nam (SHBVN)"},

	// CBBank (Xây dựng Việt Nam)
	{"CBB", "Xây dựng Việt Nam (CBB)"},
	{"CBBANK", "Xây dựng Việt Nam (CBB)"},
	{"XAYDUNG", "Xây dựng Việt Nam (CBB)"},

	// BVBank
	{"VCCB", "BVBank – Ngân hàng TMCP Bản Việt (VCCB)"},
	{"BVB", "Bảo Việt (BVB)"},

	// NCB (Quốc Dân)
	{"NCB", "Quốc Dân (NCB)"},
	{"QUOCDAN", "Quốc Dân (NCB)"},

	// LPBank (Lộc Phát Việt Nam)
	{"LPB", "Ngân hàng Thương mại Cổ phần Lộc Phát Việt Nam (LPB)"},
	{"LPBANK", "Ngân hàng Thương mại Cổ phần Lộc Phát Việt Nam (LPB)"},

	// MBV (Ngân hàng TNHH MTV Việt Nam Hiện Đại)
	{"MBV", "Ngân hàng TNHH MTV Việt Nam Hiện Đại (MBV)"},

	// VID (Liên doanh VID Public Bank)
	{"VID", "Liên doanh VID Public Bank (VID)"},

	// GP Bank (Dầu khí toàn cầu)
	{"GPB", "Dầu khí toàn cầu (GPB)"},

	// VAB (Việt Á)
	{"VAB", "Việt Á (VAB)"},
	{"VIETA", "Việt Á (VAB)"},

	// SEAB (Đông Nam Á)
	{"SEAB", "Đông Nam Á (SEAB)"},
	{"DONGNAMA", "Đông Nam Á (SEAB)"},

	// NASB (Bắc Á)
	{"NASB", "Bắc Á (NASB)"},
	{"BACA", "Bắc Á (NASB)"},

	// VietBank (Việt Nam Thương Tín)
	{"VIETBANK", "Việt Nam Thương tín (VIETBANK)"},
	{"THUONGTIN", "Việt Nam Thương tín (VIETBANK)"},

	// HLB (TNHH MTV Hong Leong VN)
	{"HLB", "TNHH MTV Hong Leong VN (HLB)"},
	{"HONGLEONG", "TNHH MTV Hong Leong VN (HLB)"},

	// CAKE by VPBank
	{"CAKE", "Việt Nam Thịnh Vượng CAKE BANK (CAKEVPB)"},

	// UBANK by VPBank
	{"UBANK", "Việt Nam Thịnh Vượng UBANK (UBANKVPB)"},
}

// sortedKeywords is keywords sorted by length descending (longest first),
// so more specific matches take priority over shorter substrings.
var sortedKeywords []Keyword

func init() {
	sortedKeywords = make([]Keyword, len(keywords))
	copy(sortedKeywords, keywords)
	sort.Slice(sortedKeywords, func(i, j int) bool {
		li, lj := len(sortedKeywords[i].Keyword), len(sortedKeywords[j].Keyword)
		if li != lj {
			return li > lj // longest first
		}
		return sortedKeywords[i].Keyword < sortedKeywords[j].Keyword
	})
}

// MapName maps a bank name from Excel/import to the actual branch_name in the banks table.
//
// Algorithm (3-pass, false-positive resistant):
//
//	Pass 1: Exact match on normalized string (uppercase, spaces removed)
//	Pass 2: Contains match for keywords ≥ 5 chars (low false-positive risk)
//	Pass 3: Word-boundary match for short keywords (< 5 chars) on original text
//
// If no match, returns the original name — which will cause resolveBankID to return nil,
// leaving the bank field empty so partners can fix it manually.
func MapName(name string) string {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return ""
	}

	normalized := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))

	// Pass 1: Exact match on normalized string
	for _, kw := range sortedKeywords {
		if normalized == kw.Keyword {
			return kw.Bank
		}
	}

	// Pass 2: Contains match for long keywords (≥ 5 chars)
	for _, kw := range sortedKeywords {
		if len(kw.Keyword) >= 5 && strings.Contains(normalized, kw.Keyword) {
			return kw.Bank
		}
	}

	// Pass 3: Word-boundary match for short keywords (< 5 chars)
	// Uses the original text (with spaces) to prevent false positives:
	//   "ACB" in "Á châu ACB" → space before, end after → match ✅
	//   "PVC" in "PV connect" → no "PVC" substring (space breaks it) → no match ✅
	upper := strings.ToUpper(raw)
	for _, kw := range sortedKeywords {
		if len(kw.Keyword) < 5 && matchAtWordBoundary(upper, kw.Keyword) {
			return kw.Bank
		}
	}

	return raw
}

// matchAtWordBoundary checks if keyword appears in text surrounded by
// non-ASCII-letter characters (or at string boundaries).
// Vietnamese characters (Â, Ê, Ơ, etc.) are treated as non-letters,
// so they act as natural word boundaries.
func matchAtWordBoundary(text, keyword string) bool {
	kl := len(keyword)
	for i := 0; i <= len(text)-kl; i++ {
		if text[i:i+kl] == keyword {
			beforeOK := i == 0 || !isASCIILetter(text[i-1])
			afterOK := i+kl >= len(text) || !isASCIILetter(text[i+kl])
			if beforeOK && afterOK {
				return true
			}
		}
	}
	return false
}

func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
