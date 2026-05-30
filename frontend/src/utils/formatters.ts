// Common formatting utilities used across all pages

const DEFAULT_CURRENCY_CODE = 'VND';

/**
 * Normalize currency codes so that UI helpers never pass invalid codes
 * to Intl.NumberFormat. Currently we only support VND, but we gracefully
 * map common Vietnamese currency symbols to the ISO code.
 */
export const normalizeCurrencyCode = (currency?: string | null): string => {
  if (!currency) return DEFAULT_CURRENCY_CODE;

  const trimmed = currency.trim();
  if (!trimmed) return DEFAULT_CURRENCY_CODE;

  const upperCased = trimmed.toUpperCase();
  if (trimmed === 'đ' || trimmed === 'đ' || upperCased === 'VNĐ') {
    return DEFAULT_CURRENCY_CODE;
  }

  return upperCased.length === 3 ? upperCased : DEFAULT_CURRENCY_CODE;
};

export const formatCurrency = (amount: number | null | undefined, currency: string = DEFAULT_CURRENCY_CODE) => {
  if (amount === null || amount === undefined || isNaN(amount)) {
    return '- đ';
  }
  const normalizedCurrency = normalizeCurrencyCode(currency);
  return new Intl.NumberFormat('vi-VN', {
    style: 'currency',
    currency: normalizedCurrency
  }).format(amount);
};

export const formatDate = (date: string | Date, format: 'short' | 'long' = 'short') => {
  const dateObj = typeof date === 'string' ? new Date(date) : date;

  if (format === 'long') {
    return new Intl.DateTimeFormat('vi-VN', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(dateObj);
  }

  return new Intl.DateTimeFormat('vi-VN').format(dateObj);
};

export const formatDateForAPI = (date: string | Date): string => {
  const dateObj = typeof date === 'string' ? new Date(date) : date;
  // Format date without timezone conversion - use local date values directly
  const year = dateObj.getFullYear();
  const month = String(dateObj.getMonth() + 1).padStart(2, '0');
  const day = String(dateObj.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

export const formatPercentage = (value: number, decimals = 1) => {
  return `${value.toFixed(decimals)}%`;
};

export const formatFileSize = (bytes: number) => {
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  if (bytes === 0) return '0 Bytes';
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
};

export const formatDuration = (minutes: number) => {
  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;

  if (hours === 0) return `${mins}m`;
  if (mins === 0) return `${hours}h`;
  return `${hours}h ${mins}m`;
};

export const getInitials = (name: string) => {
  return name
    .split(' ')
    .slice(-2)
    .map(n => n[0])
    .join('')
    .toUpperCase();
};

export const truncateText = (text: string, maxLength: number) => {
  if (text.length <= maxLength) return text;
  return text.substring(0, maxLength) + '...';
};

export const formatNumber = (value: number, decimals = 1) => {
  return new Intl.NumberFormat('vi-VN', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  }).format(value);
};
