import { format } from "date-fns";
export const getInitials = (name: string): string => {
  if (!name) return '';
  return name
    .split(' ')
    .slice(-2)
    .map(n => n[0])
    .join('')
    .toUpperCase();
};

export const formatDate = (dateString: string): string => {
  try {
    const [day, month, year] = dateString.split('/');
    const date = new Date(parseInt(year), parseInt(month) - 1, parseInt(day));
    return format(date, 'dd/MM/yyyy');
  } catch {
    return dateString;
  }
};

export const getTaskColors = () => ({
  danger: 'text-destructive bg-destructive/10',
  warning: 'text-warning bg-warning/10',
  info: 'text-info bg-info/10'
});

export const getActivityColors = () => ({
  success: 'bg-success/10',
  warning: 'bg-warning/10',
  error: 'bg-destructive/10',
  info: 'bg-info/10'
});