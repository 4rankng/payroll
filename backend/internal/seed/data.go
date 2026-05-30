package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	mathrand "math/rand/v2"
	"strings"
	"time"

	"api-server/internal/domain"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserProfile represents a complete user profile for generation
type UserProfile struct {
	FullName    string
	Username    string
	Email       string
	Phone       string
	Address     string
	CCCD        string
	BankAccount string
}

// Vietnamese data generators with weighted frequencies
var vietnameseFirstNames = map[string]int{
	"Nguyễn": 40, "Trần": 11, "Lê": 9, "Phạm": 7, "Hoàng": 4, "Huỳnh": 4,
	"Phan": 3, "Vũ": 3, "Võ": 3, "Đặng": 2, "Bùi": 2, "Đỗ": 2,
	"Hồ": 2, "Ngô": 2, "Dương": 2, "Lý": 2, "Mai": 2, "Đinh": 2, "Vương": 2,
	"Chu": 1, "Cao": 1, "Trương": 1, "Tô": 1, "Lưu": 1, "Hà": 1,
}

var vietnameseLastNames = []string{
	"Văn Minh", "Thị Hương", "Văn Hùng", "Thị Lan", "Quốc Dũng", "Thị Mai",
	"Văn Thành", "Thị Linh", "Đức Anh", "Thị Nga", "Văn Long", "Thị Thu",
	"Quang Huy", "Thị Vân", "Văn Khoa", "Thị Hoa", "Đình Trọng", "Thị Xuân",
	"Văn Tài", "Thị Bích", "Quốc Việt", "Thị Phương", "Văn Đức", "Thị Trang",
	"Minh Tuấn", "Thị Yến", "Văn Nam", "Thị Thùy", "Đức Thắng", "Thị Loan",
	"Thanh Huyền", "Anh Tuấn", "Thị Kim", "Văn Hải", "Quang Minh", "Thị Nhung",
	"Đức Long", "Thị Hạnh", "Văn Quý", "Thị Tuyết", "Minh Đức", "Thị Hòa",
}

var VietnameseCompanies = []string{
	"Công ty TNHH Sản xuất Hóa mỹ phẩm VERICO",
	"Công ty Cổ phần Dệt may Việt Nam",
	"Công ty TNHH Sản xuất Giày dép Thành Công",
	"Công ty Cổ phần Thực phẩm Sạch Việt",
	"Công ty TNHH Sản xuất Đồ gỗ Nội thất",
	"Công ty Cổ phần Điện tử Việt Nam",
	"Công ty TNHH Sản xuất Nhựa Thái Bình",
	"Công ty Cổ phần Dược phẩm Hà Nội",
}

var VietnameseBanks = []struct {
	Name string
}{
	{"Ngân hàng TMCP Ngoại thương Việt Nam - Chi nhánh Hà Nội"},
	{"Ngân hàng Đầu tư và Phát triển Việt Nam - Chi nhánh TPHCM"},
	{"Ngân hàng Nông nghiệp và Phát triển Nông thôn - Chi nhánh Đống Đa"},
	{"Ngân hàng TMCP Kỹ thương Việt Nam - Chi nhánh Cầu Giấy"},
	{"Ngân hàng TMCP Sài Gòn Thương Tín - Chi nhánh Hoàn Kiếm"},
	{"Ngân hàng TMCP Á Châu - Chi nhánh Tây Hồ"},
	{"Ngân hàng TMCP Quân đội - Chi nhánh Ba Đình"},
	{"Ngân hàng TMCP Việt Nam Thịnh Vượng - Chi nhánh Hai Bà Trưng"},
}

func GenerateVietnameseName() string {
	// Generate weighted random first name
	totalWeight := 0
	for _, weight := range vietnameseFirstNames {
		totalWeight += weight
	}

	randomValue := mathrand.IntN(totalWeight)
	currentWeight := 0
	var firstName string

	for name, weight := range vietnameseFirstNames {
		currentWeight += weight
		if randomValue < currentWeight {
			firstName = name
			break
		}
	}

	lastName := vietnameseLastNames[mathrand.IntN(len(vietnameseLastNames))]
	return firstName + " " + lastName
}

func GenerateCCCD() string {
	// Generate 12-digit CCCD starting with valid province codes
	provinceCodes := []string{"001", "002", "004", "006", "008", "010", "011", "012"}
	provinceCode := provinceCodes[mathrand.IntN(len(provinceCodes))]

	// Generate remaining 9 digits
	remaining := fmt.Sprintf("%09d", mathrand.IntN(1000000000))
	return provinceCode + remaining
}

func GenerateBankAccount() string {
	return gofakeit.CreditCardNumber(&gofakeit.CreditCardOptions{
		Types: []string{"visa"},
		Gaps:  false,
	})[0:12]
}

func GenerateVietnameseAddress() string {
	districts := []string{
		"Quận Ba Đình, Hà Nội",
		"Quận Hoàn Kiếm, Hà Nội",
		"Quận Hai Bà Trưng, Hà Nội",
		"Quận Đống Đa, Hà Nội",
		"Quận Tây Hồ, Hà Nội",
		"Quận Cầu Giấy, Hà Nội",
		"Quận 1, Thành phố Hồ Chí Minh",
		"Quận 3, Thành phố Hồ Chí Minh",
		"Quận Bình Thạnh, Thành phố Hồ Chí Minh",
		"Quận Tân Bình, Thành phố Hồ Chí Minh",
	}

	streetNumber := gofakeit.Number(1, 999)
	streetNames := []string{
		"đường Nguyễn Trãi", "đường Lê Duẩn", "đường Hai Bà Trưng",
		"đường Lý Thường Kiệt", "đường Nguyễn Huệ", "đường Điện Biên Phủ",
		"đường Trần Hưng Đạo", "đường Lê Lợi", "đường Nguyễn Du",
	}

	street := gofakeit.RandomString(streetNames)
	district := gofakeit.RandomString(districts)

	return fmt.Sprintf("%d %s, %s", streetNumber, street, district)
}

func GenerateRealisticEmail(name string) string {
	domains := []string{"company.vn", "gmail.com", "yahoo.com", "hotmail.com", "outlook.com"}
	emailDomain := gofakeit.RandomString(domains)

	nameSlug := strings.ToLower(strings.ReplaceAll(name, " ", ""))
	nameSlug = strings.ReplaceAll(nameSlug, "ă", "a")
	nameSlug = strings.ReplaceAll(nameSlug, "â", "a")
	nameSlug = strings.ReplaceAll(nameSlug, "đ", "d")
	nameSlug = strings.ReplaceAll(nameSlug, "ê", "e")
	nameSlug = strings.ReplaceAll(nameSlug, "ô", "o")
	nameSlug = strings.ReplaceAll(nameSlug, "ơ", "o")
	nameSlug = strings.ReplaceAll(nameSlug, "ư", "u")

	if gofakeit.Bool() {
		return fmt.Sprintf("%s%d@%s", nameSlug, gofakeit.Number(1, 999), emailDomain)
	}
	return fmt.Sprintf("%s@%s", nameSlug, emailDomain)
}

func GenerateRealisticPhone() string {
	prefixes := []string{"09", "08", "07", "03"}
	prefix := gofakeit.RandomString(prefixes)
	return fmt.Sprintf("%s%08d", prefix, gofakeit.Number(10000000, 99999999))
}

func GenerateCompanyName() string {
	if gofakeit.Bool() {
		return gofakeit.RandomString(VietnameseCompanies)
	}
	return gofakeit.Company()
}

func GenerateProjectName() string {
	buzzwords := []string{
		"phát triển", "sản xuất", "xây dựng", "nghiên cứu", "thiết kế",
		"marketing", "bán hàng", "quản lý", "vận hành", "cải tiến",
	}
	products := []string{
		"ứng dụng di động", "website thương mại", "hệ thống CRM", "phần mềm quản lý",
		"sản phẩm công nghệ", "giải pháp số", "nền tảng trực tuyến", "hệ thống tự động",
	}

	if gofakeit.Bool() {
		buzzword := gofakeit.RandomString(buzzwords)
		product := gofakeit.RandomString(products)
		return fmt.Sprintf("Dự án %s %s", buzzword, product)
	}
	return fmt.Sprintf("Project %s", gofakeit.BuzzWord())
}

func GenerateJobTitle() string {
	vietnameseRoles := []string{
		"Lập trình viên", "Thiết kế đồ họa", "Quản lý dự án", "Nhân viên kinh doanh",
		"Kế toán", "Nhân sự", "Marketing", "Bảo vệ", "Lao công", "Tài xế",
	}

	if gofakeit.Bool() {
		return gofakeit.RandomString(vietnameseRoles)
	}
	return gofakeit.JobTitle()
}

func GenerateWorkingHours() float64 {
	baseHours := 8.0
	variation := (gofakeit.Float64() - 0.5) * 2.0
	hours := baseHours + variation
	if hours < 0.5 {
		hours = 0.5
	}
	if hours > 12.0 {
		hours = 12.0
	}
	return float64(int(hours*100)) / 100
}

func GenerateUniqueUsername(prefix string, existingUsernames map[string]bool) string {
	for i := 1; i <= 1000; i++ {
		username := fmt.Sprintf("%s%d", prefix, i)
		if !existingUsernames[username] {
			existingUsernames[username] = true
			return username
		}
	}
	// Fallback to random generation if all numbered options are taken
	for {
		randomSuffix := gofakeit.Number(10000, 99999)
		username := fmt.Sprintf("%s%d", prefix, randomSuffix)
		if !existingUsernames[username] {
			existingUsernames[username] = true
			return username
		}
	}
}

func GenerateUniqueEmail(name string, existingEmails map[string]bool) string {
	domains := []string{"company.vn", "gmail.com", "yahoo.com", "hotmail.com", "outlook.com"}

	nameSlug := strings.ToLower(strings.ReplaceAll(name, " ", ""))
	nameSlug = strings.ReplaceAll(nameSlug, "ă", "a")
	nameSlug = strings.ReplaceAll(nameSlug, "â", "a")
	nameSlug = strings.ReplaceAll(nameSlug, "đ", "d")
	nameSlug = strings.ReplaceAll(nameSlug, "ê", "e")
	nameSlug = strings.ReplaceAll(nameSlug, "ô", "o")
	nameSlug = strings.ReplaceAll(nameSlug, "ơ", "o")
	nameSlug = strings.ReplaceAll(nameSlug, "ư", "u")

	emailDomain := gofakeit.RandomString(domains)

	// Try basic email first
	baseEmail := fmt.Sprintf("%s@%s", nameSlug, emailDomain)
	if !existingEmails[baseEmail] {
		existingEmails[baseEmail] = true
		return baseEmail
	}

	// Try with numbers
	for i := 1; i <= 999; i++ {
		email := fmt.Sprintf("%s%d@%s", nameSlug, i, emailDomain)
		if !existingEmails[email] {
			existingEmails[email] = true
			return email
		}
	}

	// Fallback to random suffix
	for {
		randomSuffix := gofakeit.Number(1000, 99999)
		email := fmt.Sprintf("%s%d@%s", nameSlug, randomSuffix, emailDomain)
		if !existingEmails[email] {
			existingEmails[email] = true
			return email
		}
	}
}

func GenerateUniqueCCCD(existingCCCDs map[string]bool) string {
	for i := 0; i < 1000; i++ {
		cccd := GenerateCCCD()
		if !existingCCCDs[cccd] {
			existingCCCDs[cccd] = true
			return cccd
		}
	}
	// Should never reach here in normal cases
	panic("unable to generate unique CCCD after 1000 attempts")
}

func GenerateUniqueProjectCode(existingCodes map[string]bool) string {
	year := clock.Now().Year() % 100

	for i := 1; i <= 9999; i++ {
		code := fmt.Sprintf("PRJ%02d%02d", year, i)
		if !existingCodes[code] {
			existingCodes[code] = true
			return code
		}
	}

	// Fallback to random generation
	for {
		randomSuffix := gofakeit.Number(1000, 9999)
		code := fmt.Sprintf("PRJ%02d%d", year, randomSuffix)
		if !existingCodes[code] {
			existingCodes[code] = true
			return code
		}
	}
}

// GenerateRealisticUserProfile creates a complete user profile with collision avoidance
func GenerateRealisticUserProfile(existingUsernames, existingEmails, existingCCCDs map[string]bool) *UserProfile {
	fullName := GenerateVietnameseName()
	return &UserProfile{
		FullName:    fullName,
		Username:    GenerateUniqueUsername("user", existingUsernames),
		Email:       GenerateUniqueEmail(fullName, existingEmails),
		Phone:       GenerateRealisticPhone(),
		Address:     GenerateVietnameseAddress(),
		CCCD:        GenerateUniqueCCCD(existingCCCDs),
		BankAccount: GenerateBankAccount(),
	}
}

// GenerateUniqueUsernameWithDB provides enhanced collision detection with database verification
func GenerateUniqueUsernameWithDB(ctx context.Context, db *gorm.DB, prefix string, existingUsernames map[string]bool) string {
	for i := 1; i <= 1000; i++ {
		username := fmt.Sprintf("%s%d", prefix, i)
		if !existingUsernames[username] {
			// Double-check against database
			var count int64
			db.Model(&domain.User{}).Where("username = ?", username).Count(&count)
			if count == 0 {
				existingUsernames[username] = true
				return username
			}
		}
	}
	// Fallback to UUID-based generation for ultimate uniqueness
	return fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(uuid.New().String()[:8], "-", ""))
}

// GenerateUniqueEmailWithDB provides enhanced collision detection with database verification
func GenerateUniqueEmailWithDB(ctx context.Context, db *gorm.DB, name string, existingEmails map[string]bool) string {
	domains := []string{"company.vn", "gmail.com", "yahoo.com", "hotmail.com", "outlook.com"}

	nameSlug := strings.ToLower(strings.ReplaceAll(name, " ", ""))
	nameSlug = strings.ReplaceAll(nameSlug, "ă", "a")
	nameSlug = strings.ReplaceAll(nameSlug, "â", "a")
	nameSlug = strings.ReplaceAll(nameSlug, "đ", "d")
	nameSlug = strings.ReplaceAll(nameSlug, "ê", "e")
	nameSlug = strings.ReplaceAll(nameSlug, "ô", "o")
	nameSlug = strings.ReplaceAll(nameSlug, "ơ", "o")
	nameSlug = strings.ReplaceAll(nameSlug, "ư", "u")

	emailDomain := gofakeit.RandomString(domains)

	// Try basic email first
	baseEmail := fmt.Sprintf("%s@%s", nameSlug, emailDomain)
	if !existingEmails[baseEmail] {
		var count int64
		db.Model(&domain.User{}).Where("email = ?", baseEmail).Count(&count)
		if count == 0 {
			existingEmails[baseEmail] = true
			return baseEmail
		}
	}

	// Try with numbers
	for i := 1; i <= 999; i++ {
		email := fmt.Sprintf("%s%d@%s", nameSlug, i, emailDomain)
		if !existingEmails[email] {
			var count int64
			db.Model(&domain.User{}).Where("email = ?", email).Count(&count)
			if count == 0 {
				existingEmails[email] = true
				return email
			}
		}
	}

	// Fallback to UUID-based generation
	return fmt.Sprintf("%s_%s@%s", nameSlug, strings.ReplaceAll(uuid.New().String()[:8], "-", ""), emailDomain)
}

// EmployeeProfile represents a complete employee profile with career progression
type EmployeeProfile struct {
	*UserProfile
	Position       string
	Experience     string
	Seniority      int // years of experience
	BasePayRate    int // VND per hour
	IsReliable     bool
	AttendanceRate float64
}

// ProjectProfile represents realistic project data
type ProjectProfile struct {
	Name             string
	ClientName       string
	Code             string
	Description      string
	Status           string
	Budget           float64
	ExpectedDuration int // months
	TeamSize         int
}

// Vietnamese position hierarchy with typical pay rates (VND per hour)
var vietnamesePositions = map[string]struct {
	BaseRate    int
	MinExp      int
	MaxExp      int
	Description string
}{
	"thực tập":       {15000, 0, 1, "Thực tập sinh - người mới bắt đầu"},
	"phổ thông":      {25000, 0, 3, "Công nhân phổ thông - không yêu cầu kinh nghiệm đặc biệt"},
	"có kinh nghiệm": {35000, 2, 8, "Công nhân có kinh nghiệm - đã quen với công việc"},
	"chuyên môn":     {45000, 3, 10, "Công nhân chuyên môn - có kỹ năng đặc biệt"},
	"tổ trưởng":      {55000, 5, 15, "Tổ trưởng - quản lý nhóm nhỏ"},
	"giám sát":       {65000, 7, 20, "Giám sát viên - quản lý ca làm việc"},
	"kỹ thuật":       {75000, 4, 15, "Kỹ thuật viên - vận hành máy móc phức tạp"},
}

// Vietnamese project types based on common manufacturing industries
var vietnameseProjectTypes = []struct {
	Industry     string
	ProjectTypes []string
	AvgBudget    float64 // million VND
	AvgDuration  int     // months
}{
	{
		"Dệt may",
		[]string{"Sản xuất áo thun", "Sản xuất quần jean", "Sản xuất đồng phục", "Dệt vải cotton"},
		2500,
		8,
	},
	{
		"Hóa mỹ phẩm",
		[]string{"Sản xuất xà phòng", "Sản xuất kem dưỡng da", "Sản xuất dầu gội", "Sản xuất nước hoa"},
		1800,
		6,
	},
	{
		"Thực phẩm",
		[]string{"Chế biến thực phẩm đông lạnh", "Sản xuất bánh kẹo", "Chế biến hải sản", "Sản xuất nước giải khát"},
		3200,
		10,
	},
	{
		"Điện tử",
		[]string{"Lắp ráp linh kiện điện tử", "Sản xuất cáp điện", "Lắp ráp thiết bị điện tử", "Sản xuất bo mạch"},
		4500,
		12,
	},
	{
		"Đồ gỗ",
		[]string{"Sản xuất nội thất", "Chế biến gỗ xuất khẩu", "Sản xuất đồ chơi gỗ", "Sản xuất đồ gia dụng"},
		2000,
		9,
	},
}

// GenerateRealisticEmployeeProfile creates a comprehensive employee profile
func GenerateRealisticEmployeeProfile(existingUsernames, existingEmails, existingCCCDs map[string]bool) *EmployeeProfile {
	baseProfile := GenerateRealisticUserProfile(existingUsernames, existingEmails, existingCCCDs)

	// Determine seniority and position based on age-like factors
	age := 20 + mathrand.IntN(40) // 20-60 years old
	seniority := max(0, age-22)   // assuming start working at 22

	// Select position based on experience
	var position string
	var baseRate int

	switch {
	case seniority <= 1:
		if mathrand.Float64() < 0.3 {
			position = "thực tập"
		} else {
			position = "phổ thông"
		}
	case seniority <= 3:
		positions := []string{"phổ thông", "có kinh nghiệm"}
		position = gofakeit.RandomString(positions)
	case seniority <= 8:
		positions := []string{"có kinh nghiệm", "chuyên môn", "tổ trưởng"}
		position = gofakeit.RandomString(positions)
	default:
		positions := []string{"chuyên môn", "tổ trưởng", "giám sát", "kỹ thuật"}
		position = gofakeit.RandomString(positions)
	}

	posInfo := vietnamesePositions[position]
	baseRate = posInfo.BaseRate

	// Add variation based on individual performance
	variation := mathrand.Float64()*0.3 - 0.15 // ±15% variation
	baseRate = int(float64(baseRate) * (1 + variation))

	// Determine reliability and attendance
	isReliable := mathrand.Float64() < 0.75 // 75% of workers are reliable
	var attendanceRate float64
	if isReliable {
		attendanceRate = 0.85 + mathrand.Float64()*0.13 // 85-98%
	} else {
		attendanceRate = 0.60 + mathrand.Float64()*0.25 // 60-85%
	}

	return &EmployeeProfile{
		UserProfile:    baseProfile,
		Position:       position,
		Experience:     posInfo.Description,
		Seniority:      seniority,
		BasePayRate:    baseRate,
		IsReliable:     isReliable,
		AttendanceRate: attendanceRate,
	}
}

// GenerateRealisticProjectProfile creates a comprehensive project profile
func GenerateRealisticProjectProfile(existingCodes map[string]bool) *ProjectProfile {
	industryData := vietnameseProjectTypes[mathrand.IntN(len(vietnameseProjectTypes))]
	projectType := gofakeit.RandomString(industryData.ProjectTypes)

	// Generate realistic client company name
	companyTypes := []string{"Công ty TNHH", "Công ty Cổ phần", "Doanh nghiệp"}
	companyType := gofakeit.RandomString(companyTypes)

	industryNames := []string{
		"Sản xuất " + strings.ToLower(industryData.Industry),
		industryData.Industry + " Việt Nam",
		industryData.Industry + " Thành Đạt",
		industryData.Industry + " Hưng Phát",
	}
	industryName := gofakeit.RandomString(industryNames)
	clientName := fmt.Sprintf("%s %s", companyType, industryName)

	// Generate budget with realistic variation
	budgetVariation := 0.5 + mathrand.Float64()                  // 0.5x to 1.5x base budget
	budget := industryData.AvgBudget * budgetVariation * 1000000 // convert to VND

	// Generate duration with variation
	durationVariation := 0.7 + mathrand.Float64()*0.6 // 0.7x to 1.3x
	duration := int(float64(industryData.AvgDuration) * durationVariation)
	if duration < 1 {
		duration = 1
	}

	// Team size based on budget and duration
	teamSize := int(budget / float64(duration) / 12 / 30 / 8 / 35000) // rough calculation based on average worker cost
	if teamSize < 5 {
		teamSize = 5 + mathrand.IntN(10)
	}
	if teamSize > 100 {
		teamSize = 50 + mathrand.IntN(50)
	}

	// Generate project status based on realistic distribution
	statusRand := mathrand.Float64()
	var status string
	switch {
	case statusRand < 0.1:
		status = "draft"
	case statusRand < 0.6:
		status = "active"
	case statusRand < 0.85:
		status = "completed"
	default:
		status = "cancelled"
	}

	description := fmt.Sprintf(
		"Dự án %s được triển khai tại %s nhằm nâng cao năng suất và chất lượng sản phẩm. "+
			"Dự kiến quy mô %d nhân viên trong %d tháng với ngân sách %.0f triệu VND.",
		projectType, clientName, teamSize, duration, budget/1000000,
	)

	return &ProjectProfile{
		Name:             projectType,
		ClientName:       clientName,
		Code:             GenerateUniqueProjectCode(existingCodes),
		Description:      description,
		Status:           status,
		Budget:           budget,
		ExpectedDuration: duration,
		TeamSize:         teamSize,
	}
}

// GenerateRealisticPayrates creates realistic pay rate structures
func GenerateRealisticPayrates(position string, baseRate int) map[string]int {
	rates := make(map[string]int)

	rates["normal"] = baseRate
	rates["overtime"] = int(float64(baseRate) * 1.5) // 150% for overtime (Vietnamese labor law)
	rates["weekend"] = int(float64(baseRate) * 2.0)  // 200% for weekend work
	rates["holiday"] = int(float64(baseRate) * 3.0)  // 300% for holiday work

	// Add some position-specific variations
	switch position {
	case "giám sát", "kỹ thuật":
		// Supervisors get slightly higher weekend/holiday premiums
		rates["weekend"] = int(float64(baseRate) * 2.2)
		rates["holiday"] = int(float64(baseRate) * 3.5)
	case "thực tập":
		// Interns get lower premiums
		rates["overtime"] = int(float64(baseRate) * 1.3)
		rates["weekend"] = int(float64(baseRate) * 1.8)
		rates["holiday"] = int(float64(baseRate) * 2.5)
	}

	return rates
}

// GenerateVietnameseHolidays returns major Vietnamese holidays for a given year
func GenerateVietnameseHolidays(year int) []time.Time {
	holidays := []time.Time{
		time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),  // New Year
		time.Date(year, 4, 30, 0, 0, 0, 0, time.UTC), // Liberation Day
		time.Date(year, 5, 1, 0, 0, 0, 0, time.UTC),  // Labor Day
		time.Date(year, 9, 2, 0, 0, 0, 0, time.UTC),  // National Day
	}

	// Add Tet holidays (approximate - 3-5 days in Jan/Feb)
	tetStart := time.Date(year, 1, 20+mathrand.IntN(20), 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		holidays = append(holidays, tetStart.AddDate(0, 0, i))
	}

	return holidays
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	_ = gofakeit.Seed(clock.Now().UnixNano())
}
