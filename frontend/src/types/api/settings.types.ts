export interface Setting {
  id: number;
  key: string;
  value: string | null;
  value_type: 'string' | 'number' | 'boolean' | 'json';
  updated_at: string;
  /**
   * Masked is true when the backend withheld the value because the key holds a
   * sealed secret (e.g. `mimo.api_key`); Configured then reports whether a
   * value is stored at all — enough to render connection status without ever
   * shipping the credential to the browser.
   */
  masked?: boolean;
  configured?: boolean;
}

export interface CreateSettingData {
  key: string;
  value?: string | null;
  value_type: 'string' | 'number' | 'boolean' | 'json';
}

export interface UpdateSettingData {
  key?: string;
  value?: string | null;
  value_type?: 'string' | 'number' | 'boolean' | 'json';
}

export interface SettingsFilters {
  page?: number;
  pageSize?: number;
  search?: string;
  value_type?: string;
  [key: string]: unknown;
}

export interface SettingsResponse {
  status: string;
  data: Setting[];
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  message: string;
}

export interface SettingResponse {
  status: string;
  data: Setting;
  message: string;
}

/**
 * MiMo test-connection outcome codes — mirror backend dto.MimoTest*. Each code
 * maps to one designed Vietnamese error state in MimoConnectionSection (which
 * buttons stay enabled is part of that design).
 */
export type MimoTestConnectionCode =
  | 'ok'
  | 'missing_key'
  | 'invalid_key'
  | 'timeout'
  | 'rate_limited'
  | 'unavailable'
  | 'busy'
  | 'error';

export interface MimoTestConnectionResult {
  success: boolean;
  code: MimoTestConnectionCode;
  message: string;
  latency_ms?: number;
  model_count?: number;
  default_model_present?: boolean;
}

/** Settings keys managed by the MiMo assistant connection section. */
export const MIMO_SETTING_KEYS = {
  apiKey: 'mimo.api_key',
  baseUrl: 'mimo.base_url',
  defaultModel: 'mimo.default_model',
  fallbackModel: 'mimo.fallback_model',
  modelChat: 'mimo.model_chat',
  modelDeliverable: 'mimo.model_deliverable',
  chatEnabledAdmin: 'mimo.chat_enabled_admin',
  chatEnabledPartner: 'mimo.chat_enabled_partner',
} as const;

/** Allowed MiMo model IDs — mirror backend llm.AllowedModels. */
export const MIMO_MODELS = ['mimo-v2.6-flash', 'mimo-v2.6-pro'] as const;
export type MimoModel = (typeof MIMO_MODELS)[number];