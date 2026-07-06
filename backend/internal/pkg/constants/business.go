package constants

// BulkTransferPaymentPercentage represents the percentage of timesheet amount paid in bulk transfers
const BulkTransferPaymentPercentage = 0.70

// Sheet names for different bank templates
const MBank_SheetName = "eMB_BulkPayment"

// Bank IDs for bulk transfer source banks
const MBankID uint = 4488

// File prefixes for bulk transfer files
const MBankPrefix = "MBank"

// Template paths for bulk transfer
const MBankTemplatePath = "templates/MBank-bulk-transfer-template.xlsx"
const PayrollReportTemplatePath = "templates/payroll_template.xlsx"
const PayrollReportByProjectTemplatePath = "templates/sao_ke_tt_theo_du_an.xlsx"
const PayrollEmailTemplatePath = "templates/email/payroll_statement.html"
const AdvancePaymentReminderTemplatePath = "templates/email/advance_payment_reminder.html"
const BrandBannerPath = "templates/email/banner_inline.jpg"
const EmployeeProfileTemplatePath = "templates/HoSoNhanSu.xlsx"

const MyCompany = "Hệ thống"
const PartnerCompany = "VFIC Manpower"

// Bulk transfer filename patterns for payment schedule detection
const CycleMonthly = "monthly"
const CycleWeekly = "weekly"
const CycleFlexible = "flexible"
