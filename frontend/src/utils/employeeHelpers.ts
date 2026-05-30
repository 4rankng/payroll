export const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat('vi-VN', {
    minimumFractionDigits: 0,
    maximumFractionDigits: 0
  }).format(amount) + ' đ';
};

export const getInitials = (name: string) => {
  if (!name || typeof name !== 'string') {
    return '';
  }

  return name
    .split(' ')
    .slice(-2)
    .map(n => n[0])
    .join('')
    .toUpperCase();
};
