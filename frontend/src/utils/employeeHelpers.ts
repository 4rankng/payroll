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
