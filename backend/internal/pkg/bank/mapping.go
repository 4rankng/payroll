package bank

import "strings"

// Keyword represents a keyword and its corresponding bank name
// The bank name must match exact branch_name value in the banks table
type Keyword struct {
	Keyword string
	Bank    string // Must match exact branch_name in database
}

// keywords contains all keyword mappings for bank name resolution
// Order matters: more specific keywords should come first
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

	// Sacombank (Sài Gòn Công thương / Sài Gòn)
	{"SACOMBANK", "Sài Gòn (STB)"},
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
	{"PGB", "Ngân hàng TMCP Thượng vượng và Phát triển (PGB)"},
	{"PGBANK", "Ngân hàng TMCP Thượng vượng và Phát triển (PGB)"},

	// OceanBank (Đại Dương)
	{"PVC", "Đại Dương (PVC)"},
	{"OCEANBANK", "Đại Dương (PVC)"},
	{"DAIDUONG", "Đại Dương (PVC)"},

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
	{"VCCB", "BVBank Việt Ngân hàng TMCP Bản Việt (VCCB)"},
	{"BVB", "Bảo Việt (BVB)"},

	// NCB (Quốc Dân)
	{"NCB", "Quốc Dân (NCB)"},
	{"QUOCDAN", "Quốc Dân (NCB)"},

	// LPB (Ngân hàng Thương mại Cổ phần Lực Phát Việt Nam)
	{"LPB", "Ngân hàng Thương mại Cổ phần Lực Phát Việt Nam (LPB)"},

	// MBV (Ngân hàng TNHH MTV Việt Nam Hiện Đại)
	{"MBV", "Ngân hàng TNHH MTV Việt Nam Hiện Đại (MBV)"},

	// VID (Liên doanh VID Public Bank)
	{"VID", "Liên doanh VID Public Bank (VID)"},

	// GP Bank (Đầu khu toán cảu)
	{"GPB", "Đầu khu toán cảu (GPB)"},

	// VAB (Việt Á)
	{"VAB", "Việt Á (VAB)"},
	{"VIETA", "Việt Á (VAB)"},

	// SEAB (Đông Nam Á)
	{"SEAB", "Đông Nam Á (SEAB)"},
	{"DONGNAMA", "Đông Nam Á (SEAB)"},

	// NASB (Bắc Á)
	{"NASB", "Bắc Á (NASB)"},
	{"BACA", "Bắc Á (NASB)"},

	// BVB (Bảo Việt)
	{"BVB", "Bảo Việt (BVB)"},

	// VietBank (Việt Nam Thương Tín)
	{"VIETBANK", "Việt Nam Thương Tín (VIETBANK)"},
	{"THUONGTIN", "Việt Nam Thương Tín (VIETBANK)"},

	// HLB (TNHH MTV Hong Leong VN)
	{"HLB", "TNHH MTV Hong Leong VN (HLB)"},
	{"HONGLEONG", "TNHH MTV Hong Leong VN (HLB)"},

	// CAKE by VPBank
	{"CAKE", "Việt Nam Thịnh Vượng CAKE BANK (CAKEVPB)"},

	// UBANK by VPBank
	{"UBANK", "Việt Nam Thịnh Vượng UBANK (UBANKVPB)"},
}

// MapName maps a bank name from Excel/import to the actual branch_name in the banks table
// It uses keyword-based matching - checks if the normalized name contains any of the keywords
//
// Example:
//
//	"Ngân hàng TMCP An Bình - ABBANK" -> "An Bình (ABBANK)"
//	"Ngân hàng TMCP Đầu tư và Phát triển Việt Nam - BIDV" -> "Đầu tư và Phát triển (BIDV)"
func MapName(name string) string {
	normalizedName := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), " ", ""))

	for _, kw := range keywords {
		if strings.Contains(normalizedName, kw.Keyword) {
			return kw.Bank
		}
	}

	return name
}
