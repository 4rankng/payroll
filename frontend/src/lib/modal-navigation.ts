/**
 * Modal Navigation Utilities
 * Simple helper functions for managing multi-level modal navigation using returnTo parameter
 */

export interface ReturnToEntry {
  modalId: string;
  params: Record<string, string>;
}

/**
 * Encode current modal state and chain it with existing returnTo
 * @param modalId Current modal ID
 * @param params Current modal parameters
 * @param existingReturnTo Previous returnTo chain (if any)
 * @returns Encoded returnTo string
 */
export function encodeReturnPath(
  modalId: string,
  params: Record<string, string> = {},
  existingReturnTo?: string
): string {
  // Create current modal entry
  const paramString = Object.entries(params)
    .filter(([key, value]) => key !== 'modal' && key !== 'returnTo' && value)
    .map(([key, value]) => `${key}=${value}`)
    .join('&');

  const currentEntry = paramString ? `${modalId}:${paramString}` : modalId;

  // Chain with existing returnTo
  if (existingReturnTo) {
    return `${currentEntry}>${existingReturnTo}`;
  }

  return currentEntry;
}

/**
 * Decode returnTo string into navigation entries
 * @param returnTo Encoded returnTo string
 * @returns Array of navigation entries (first is immediate parent)
 */
export function decodeReturnPath(returnTo: string): ReturnToEntry[] {
  if (!returnTo) return [];

  return returnTo.split('>').map(entry => {
    const [modalId, paramString] = entry.split(':');
    const params: Record<string, string> = {};

    if (paramString) {
      paramString.split('&').forEach(pair => {
        const [key, value] = pair.split('=');
        if (key && value) {
          params[key] = decodeURIComponent(value);
        }
      });
    }

    return { modalId, params };
  });
}

/**
 * Get the immediate parent modal from returnTo string
 * @param returnTo Encoded returnTo string
 * @returns Parent modal entry or null
 */
export function getParentModal(returnTo: string): ReturnToEntry | null {
  const entries = decodeReturnPath(returnTo);
  return entries.length > 0 ? entries[0] : null;
}

/**
 * Get the remaining returnTo chain after removing the first parent
 * @param returnTo Encoded returnTo string
 * @returns Remaining chain or empty string
 */
export function getRemainingReturnPath(returnTo: string): string {
  const parts = returnTo.split('>');
  return parts.slice(1).join('>');
}

/**
 * Build search parameters with modal and returnTo
 * @param modalId Modal to open
 * @param modalParams Parameters for the modal
 * @param returnTo Return path
 * @returns URLSearchParams object
 */
export function buildModalSearchParams(
  modalId: string,
  modalParams: Record<string, string | number | boolean> = {},
  returnTo?: string
): URLSearchParams {
  const searchParams = new URLSearchParams();
  searchParams.set('modal', modalId);

  // Add modal parameters
  Object.entries(modalParams).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.set(key, String(value));
    }
  });

  // Add returnTo if provided
  if (returnTo) {
    searchParams.set('returnTo', returnTo);
  }

  return searchParams;
}

/**
 * Extract current modal state from URL search params
 * @param searchParams Current URL search parameters
 * @returns Current modal state
 */
export function extractCurrentModalState(searchParams: URLSearchParams): {
  modalId: string | null;
  params: Record<string, string>;
  returnTo: string | null;
} {
  const modalId = searchParams.get('modal');
  const returnTo = searchParams.get('returnTo');
  const params: Record<string, string> = {};

  // Extract all params except modal and returnTo
  searchParams.forEach((value, key) => {
    if (key !== 'modal' && key !== 'returnTo') {
      params[key] = value;
    }
  });

  return { modalId, params, returnTo };
}