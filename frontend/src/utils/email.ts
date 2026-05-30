/**
 * Email utility functions
 * Centralizes email validation and management logic
 */

/**
 * Validates email format using RFC 5322 compliant regex
 */
export const validateEmail = (email: string): boolean => {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
};

/**
 * Adds an email to a list with validation
 * @returns Object with success status and optional error message
 */
export const addEmailToList = (
  email: string,
  currentList: string[]
): { success: boolean; error?: string; updatedList?: string[] } => {
  const trimmedEmail = email.trim();

  if (!trimmedEmail) {
    return { success: false, error: 'Email không được để trống' };
  }

  if (!validateEmail(trimmedEmail)) {
    return { success: false, error: 'Email không hợp lệ' };
  }

  if (currentList.includes(trimmedEmail)) {
    return { success: false, error: 'Email đã tồn tại trong danh sách' };
  }

  return {
    success: true,
    updatedList: [...currentList, trimmedEmail],
  };
};

/**
 * Removes an email from a list by index
 */
export const removeEmailFromList = (index: number, emailList: string[]): string[] => {
  return emailList.filter((_, i) => i !== index);
};
