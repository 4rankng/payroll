import { stripVietnameseDiacritics } from '@/utils/vietnameseNormalization';

// Validation utilities for Vietnamese business requirements

// CCCD (Citizen ID) validation - 12 digits
export const validateCCCD = (cccd: string): { valid: boolean; error?: string } => {
  const cleanCCCD = cccd.replace(/\s/g, '');

  if (!cleanCCCD) {
    return { valid: false, error: 'CCCD là bắt buộc' };
  }

  if (!/^\d{12}$/.test(cleanCCCD)) {
    return { valid: false, error: 'CCCD phải có đúng 12 chữ số' };
  }

  // Validate province code (first 3 digits)
  const provinceCode = parseInt(cleanCCCD.substring(0, 3));
  const validProvinceCodes = [
    1, 2, 4, 6, 8, 10, 11, 12, 14, 15, 17, 19, 20, 22, 24, 25, 26, 27,
    30, 31, 33, 34, 35, 36, 37, 38, 40, 42, 44, 45, 46, 48, 49, 51, 52,
    54, 56, 58, 60, 62, 64, 66, 67, 68, 70, 72, 74, 75, 77, 79, 80, 82,
    83, 84, 86, 87, 89, 91, 92, 93, 94, 95, 96
  ];

  if (!validProvinceCodes.includes(provinceCode)) {
    return { valid: false, error: 'Mã tỉnh/thành phố không hợp lệ' };
  }

  return { valid: true };
};

// Vietnamese phone number validation
export const validatePhoneNumber = (phone: string): { valid: boolean; error?: string } => {
  const cleanPhone = phone.replace(/[\s.-]/g, '');

  if (!cleanPhone) {
    return { valid: false, error: 'Số điện thoại là bắt buộc' };
  }

  // Vietnamese phone patterns
  const patterns = [
    /^(84|\+84|0)(3[2-9]|5[689]|7[06-9]|8[1-689]|9[0-46-9])\d{7}$/, // Mobile
    /^(84|\+84|0)(2[0-9]{1,2})\d{7,8}$/ // Landline
  ];

  const isValid = patterns.some(pattern => pattern.test(cleanPhone));

  if (!isValid) {
    return { valid: false, error: 'Số điện thoại không hợp lệ' };
  }

  return { valid: true };
};

/**
 * A 12-digit value in a phone field is a CCCD (citizen ID), never a mobile
 * number: 12 digits is longer than any Vietnamese mobile, even in 84 form.
 * The phone column is often swapped with the CCCD column in partner sheets.
 */
export const isCCCDValue = (value: string): boolean =>
  /^\d{12}$/.test(value.replace(/[\s.-]/g, ''));

/** Guards the phone field against a CCCD typed/imported into it. */
export const validateMobileNotCCCD = (mobile: string): { valid: boolean; error?: string } => {
  if (mobile && isCCCDValue(mobile)) {
    return { valid: false, error: 'Số điện thoại không được là số CCCD (12 chữ số)' };
  }
  return { valid: true };
};
export const validateEmail = (email: string): { valid: boolean; error?: string } => {
  if (!email) {
    return { valid: false, error: 'Email là bắt buộc' };
  }

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  if (!emailRegex.test(email)) {
    return { valid: false, error: 'Email không hợp lệ' };
  }

  return { valid: true };
};

// Date validation (DD/MM/YYYY format)
export const validateDate = (dateStr: string): { valid: boolean; error?: string } => {
  if (!dateStr) {
    return { valid: false, error: 'Ngày là bắt buộc' };
  }

  const dateRegex = /^(\d{2})\/(\d{2})\/(\d{4})$/;
  const match = dateStr.match(dateRegex);

  if (!match) {
    return { valid: false, error: 'Định dạng ngày phải là DD/MM/YYYY' };
  }

  const [, day, month, year] = match;
  const date = new Date(parseInt(year), parseInt(month) - 1, parseInt(day));

  if (
    date.getDate() !== parseInt(day) ||
    date.getMonth() !== parseInt(month) - 1 ||
    date.getFullYear() !== parseInt(year)
  ) {
    return { valid: false, error: 'Ngày không hợp lệ' };
  }

  return { valid: true };
};

// Date range validation
export const validateDateRange = (
  startDate: string,
  endDate: string
): { valid: boolean; error?: string } => {
  const startValidation = validateDate(startDate);
  if (!startValidation.valid) {
    return { valid: false, error: `Ngày bắt đầu: ${startValidation.error}` };
  }

  const endValidation = validateDate(endDate);
  if (!endValidation.valid) {
    return { valid: false, error: `Ngày kết thúc: ${endValidation.error}` };
  }

  const [startDay, startMonth, startYear] = startDate.split('/');
  const [endDay, endMonth, endYear] = endDate.split('/');

  const start = new Date(parseInt(startYear), parseInt(startMonth) - 1, parseInt(startDay));
  const end = new Date(parseInt(endYear), parseInt(endMonth) - 1, parseInt(endDay));

  if (end <= start) {
    return { valid: false, error: 'Ngày kết thúc phải sau ngày bắt đầu' };
  }

  return { valid: true };
};

// Working hours validation (0-24)
export const validateWorkingHours = (hours: number): { valid: boolean; error?: string } => {
  if (hours === undefined || hours === null) {
    return { valid: false, error: 'Số giờ là bắt buộc' };
  }

  if (isNaN(hours)) {
    return { valid: false, error: 'Số giờ phải là số' };
  }

  if (hours < 0 || hours > 24) {
    return { valid: false, error: 'Số giờ phải từ 0 đến 24' };
  }

  return { valid: true };
};

// Currency validation (VND, no decimals)
export const validateCurrency = (amount: number | string): { valid: boolean; error?: string } => {
  const numAmount = typeof amount === 'string' ? parseInt(amount.replace(/[.,]/g, '')) : amount;

  if (isNaN(numAmount)) {
    return { valid: false, error: 'Số tiền không hợp lệ' };
  }

  if (numAmount < 0) {
    return { valid: false, error: 'Số tiền không thể âm' };
  }

  if (!Number.isInteger(numAmount)) {
    return { valid: false, error: 'Số tiền phải là số nguyên (₫)' };
  }

  return { valid: true };
};

// Format currency for display — delegates to canonical implementation
import { formatCurrency as _formatCurrency } from '@/utils/formatters';
export { _formatCurrency as formatCurrency };

// Format currency in full form with the ₫ symbol
export const formatCurrencyShort = (amount: number): string => {
  return _formatCurrency(amount);
};

// Project code validation (PRJ-YYYY-XXX format)
export const validateProjectCode = (code: string): { valid: boolean; error?: string } => {
  if (!code) {
    return { valid: false, error: 'Mã dự án là bắt buộc' };
  }

  const codeRegex = /^PRJ-\d{4}-\d{3}$/;

  if (!codeRegex.test(code)) {
    return { valid: false, error: 'Mã dự án phải theo định dạng PRJ-YYYY-XXX' };
  }

  return { valid: true };
};

// Password strength validation
export const validatePassword = (password: string): {
  valid: boolean;
  strength: 'weak' | 'medium' | 'strong';
  errors: string[];
} => {
  const errors: string[] = [];
  let strength: 'weak' | 'medium' | 'strong' = 'weak';

  if (!password) {
    return { valid: false, strength, errors: ['Mật khẩu là bắt buộc'] };
  }

  if (password.length < 8) {
    errors.push('Mật khẩu phải có ít nhất 8 ký tự');
  }

  if (!/[A-Z]/.test(password)) {
    errors.push('Mật khẩu phải có ít nhất 1 chữ hoa');
  }

  if (!/[a-z]/.test(password)) {
    errors.push('Mật khẩu phải có ít nhất 1 chữ thường');
  }

  if (!/[0-9]/.test(password)) {
    errors.push('Mật khẩu phải có ít nhất 1 số');
  }

  if (!/[!@#$%^&*(),.?":{}|<>]/.test(password)) {
    errors.push('Mật khẩu phải có ít nhất 1 ký tự đặc biệt');
  }

  // Calculate strength
  const hasLength = password.length >= 8;
  const hasUpperCase = /[A-Z]/.test(password);
  const hasLowerCase = /[a-z]/.test(password);
  const hasNumber = /[0-9]/.test(password);
  const hasSpecialChar = /[!@#$%^&*(),.?":{}|<>]/.test(password);

  const strengthScore = [
    hasLength,
    hasUpperCase,
    hasLowerCase,
    hasNumber,
    hasSpecialChar
  ].filter(Boolean).length;

  if (strengthScore === 5) {
    strength = 'strong';
  } else if (strengthScore >= 3) {
    strength = 'medium';
  }

  return {
    valid: errors.length === 0,
    strength,
    errors
  };
};

// Format Vietnamese name to title case
// Removes numbers, special characters, normalizes whitespace, and capitalizes first letter of each word
export const formatVietnameseName = (name: string): string => {
  if (!name) {
    return '';
  }

  // Helper function to check if a character is a letter (including Vietnamese characters)
  const isLetter = (char: string): boolean => {
    const code = char.charCodeAt(0);
    return (
      (code >= 65 && code <= 90) ||   // A-Z
      (code >= 97 && code <= 122) ||  // a-z
      code > 127                       // Vietnamese and other Unicode letters
    );
  };

  // Step 1: Remove all numbers and special characters, keep only letters and spaces
  let cleaned = '';
  for (const char of name) {
    if (isLetter(char) || char === ' ') {
      cleaned += char;
    }
  }

  // Step 2: Trim and normalize whitespace (remove leading/trailing spaces, collapse multiple spaces)
  cleaned = cleaned.trim().replace(/\s+/g, ' ');

  // Step 3: Return empty string if nothing left after cleaning
  if (!cleaned) {
    return '';
  }

  // Step 4: Convert to title case (capitalize first letter of each word)
  const words = cleaned.split(' ');
  const titleCased = words.map(word => {
    if (!word) return '';
    return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
  }).filter(word => word !== ''); // Remove any empty strings

  return titleCased.join(' ');
};

// Full name validation (letters and spaces only, including Vietnamese characters)
export const validateFullname = (fullname: string): { valid: boolean; error?: string } => {
  const trimmedName = fullname?.trim();

  if (!trimmedName) {
    return { valid: false, error: 'Họ tên là bắt buộc' };
  }

  // Normalize Vietnamese characters (NFD) and remove diacritics to get base characters
  // Then check if base characters are only letters and spaces
  const normalized = stripVietnameseDiacritics(trimmedName);

  const nameRegex = /^[a-zA-Z\s]+$/;

  if (!nameRegex.test(normalized)) {
    return { valid: false, error: 'Họ tên chỉ được chứa chữ cái và khoảng trắng' };
  }

  return { valid: true };
};

// Bank account validation (Vietnamese bank account)
export const validateBankAccount = (accountNumber: string): { valid: boolean; error?: string } => {
  const cleanAccount = accountNumber.replace(/\s/g, '');

  if (!cleanAccount) {
    return { valid: false, error: 'Số tài khoản là bắt buộc' };
  }

  // Vietnamese bank accounts typically have 9-14 digits
  if (!/^\d{9,14}$/.test(cleanAccount)) {
    return { valid: false, error: 'Số tài khoản phải có từ 9 đến 14 chữ số' };
  }

  return { valid: true };
};

// SWIFT code validation
export const validateSWIFTCode = (code: string): { valid: boolean; error?: string } => {
  if (!code) {
    return { valid: false, error: 'Mã SWIFT là bắt buộc' };
  }

  // SWIFT code format: 8 or 11 characters
  if (!/^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$/.test(code)) {
    return { valid: false, error: 'Mã SWIFT không hợp lệ' };
  }

  return { valid: true };
};

// Duplicate check helper
export const checkDuplicate = <T>(
  value: T,
  existingValues: T[],
  fieldName: string
): { valid: boolean; error?: string } => {
  if (existingValues.includes(value)) {
    return { valid: false, error: `${fieldName} đã tồn tại` };
  }

  return { valid: true };
};

// Form validation helper
export const validateForm = (
  formData: Record<string, unknown>,
  validations: Record<string, (value: unknown) => { valid: boolean; error?: string }>
): { valid: boolean; errors: Record<string, string> } => {
  const errors: Record<string, string> = {};

  for (const [field, validator] of Object.entries(validations)) {
    const result = validator(formData[field]);
    if (!result.valid && result.error) {
      errors[field] = result.error;
    }
  }

  return {
    valid: Object.keys(errors).length === 0,
    errors
  };
};
