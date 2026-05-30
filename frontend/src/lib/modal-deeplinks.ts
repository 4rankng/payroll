import type { ModalId, ModalParams } from '@/types/modal-config.types';
import { getModalConfig } from './modal-registry';

/**
 * Deeplink Utilities for Type-Safe URL Building
 * Provides utilities for creating and validating modal deeplinks
 */

/**
 * Build a deeplink URL for a modal with parameters
 */
export async function buildDeeplink(
  modalId: ModalId,
  params?: ModalParams,
  options?: {
    baseUrl?: string;
    includeProtocol?: boolean;
  }
): Promise<string> {
  const modalConfig = await getModalConfig(modalId);
  
  if (!modalConfig) {
    throw new Error(`Modal not found in registry: ${modalId}`);
  }

  if (!modalConfig.deeplink.enabled) {
    throw new Error(`Deeplinks not enabled for modal: ${modalId}`);
  }

  // Validate parameters if validator exists
  if (params && modalConfig.deeplink.validateParams) {
    if (!modalConfig.deeplink.validateParams(params)) {
      throw new Error(`Invalid parameters for modal deeplink: ${modalId}`);
    }
  }

  // Build URL parameters
  const searchParams = new URLSearchParams();
  searchParams.set('modal', modalId);

  // Add modal-specific parameters
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        searchParams.set(key, String(value));
      }
    });
  }

  const queryString = searchParams.toString();
  const { baseUrl = '', includeProtocol = false } = options || {};
  
  if (includeProtocol || baseUrl.includes('://')) {
    return `${baseUrl}?${queryString}`;
  } else {
    const path = baseUrl || window.location.pathname;
    return `${path}?${queryString}`;
  }
}

/**
 * Synchronous deeplink builder (for components that can't use async)
 * Note: Doesn't validate modal config, use with caution
 */
export function buildDeeplinkSync(
  modalId: ModalId,
  params?: ModalParams,
  baseUrl?: string
): string {
  const searchParams = new URLSearchParams();
  searchParams.set('modal', modalId);

  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        searchParams.set(key, String(value));
      }
    });
  }

  const queryString = searchParams.toString();
  const path = baseUrl || window.location.pathname;
  return `${path}?${queryString}`;
}

/**
 * Parse deeplink parameters from URL
 */
export function parseDeeplink(url?: string): {
  modalId: ModalId | null;
  params: ModalParams;
} {
  const urlToParse = url || window.location.search;
  const searchParams = new URLSearchParams(urlToParse);
  
  const modalId = searchParams.get('modal');
  const params: ModalParams = {};

  // Extract all parameters except 'modal'
  searchParams.forEach((value, key) => {
    if (key !== 'modal') {
      // Try to parse as number or boolean, fallback to string
      if (value === 'true') {
        params[key] = true;
      } else if (value === 'false') {
        params[key] = false;
      } else if (!isNaN(Number(value)) && value !== '') {
        params[key] = Number(value);
      } else {
        params[key] = value;
      }
    }
  });

  return { modalId, params };
}

/**
 * Validate deeplink against modal configuration
 */
export async function validateDeeplink(
  modalId: ModalId,
  params: ModalParams
): Promise<{
  valid: boolean;
  errors: string[];
}> {
  const errors: string[] = [];
  
  try {
    const modalConfig = await getModalConfig(modalId);
    
    if (!modalConfig) {
      errors.push(`Modal not found: ${modalId}`);
      return { valid: false, errors };
    }

    if (!modalConfig.deeplink.enabled) {
      errors.push(`Deeplinks not enabled for modal: ${modalId}`);
    }

    // Validate expected parameters
    if (modalConfig.deeplink.params) {
      modalConfig.deeplink.params.forEach(expectedParam => {
        if (!(expectedParam in params)) {
          errors.push(`Missing required parameter: ${expectedParam}`);
        }
      });
    }

    // Run custom validation
    if (modalConfig.deeplink.validateParams) {
      if (!modalConfig.deeplink.validateParams(params)) {
        errors.push(`Custom parameter validation failed`);
      }
    }

  } catch (error) {
    errors.push(`Validation error: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Generate deeplink documentation for a modal
 */
export async function getDeeplinkDocs(modalId: ModalId): Promise<{
  example: string;
  params: string[];
  description: string;
} | null> {
  const modalConfig = await getModalConfig(modalId);
  
  if (!modalConfig || !modalConfig.deeplink.enabled) {
    return null;
  }

  return {
    example: modalConfig.deeplink.example || buildDeeplinkSync(modalId, {}),
    params: modalConfig.deeplink.params || [],
    description: modalConfig.description || `Deeplink for ${modalConfig.name || modalId}`
  };
}

/**
 * Type-safe deeplink builders for common patterns
 */
export const deeplinkBuilders = {
  /**
   * Build deeplink for modals with ID parameter
   */
  withId: (modalId: ModalId, id: string | number, additionalParams?: ModalParams) => {
    return buildDeeplinkSync(modalId, { id, ...additionalParams });
  },

  /**
   * Build deeplink for modals with tab parameter
   */
  withTab: (modalId: ModalId, tab: string, additionalParams?: ModalParams) => {
    return buildDeeplinkSync(modalId, { tab, ...additionalParams });
  },

  /**
   * Build deeplink for modals with ID and tab
   */
  withIdAndTab: (modalId: ModalId, id: string | number, tab: string, additionalParams?: ModalParams) => {
    return buildDeeplinkSync(modalId, { id, tab, ...additionalParams });
  }
};

/**
 * React hook for deeplink building (for components)
 */
export function useDeeplinkBuilder() {
  const buildLink = async (modalId: ModalId, params?: ModalParams) => {
    return buildDeeplink(modalId, params);
  };

  const buildLinkSync = (modalId: ModalId, params?: ModalParams) => {
    return buildDeeplinkSync(modalId, params);
  };

  const parseCurrentUrl = () => {
    return parseDeeplink();
  };

  return {
    buildLink,
    buildLinkSync,
    parseCurrentUrl,
    builders: deeplinkBuilders
  };
}