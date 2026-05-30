export interface Setting {
  id: number;
  key: string;
  value: string | null;
  value_type: 'string' | 'number' | 'boolean' | 'json';
  updated_at: string;
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