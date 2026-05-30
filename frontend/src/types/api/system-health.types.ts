export interface APILatency {
  p95: number;
  min: number;
  max: number;
  avg: number;
  median: number;
}

export interface APIErrorDetail {
  total: number;
  [statusCode: string]: number;
}

export interface APISummaryItem {
  endpoint: string;
  count: number;
  latency: APILatency;
  success: number;
  error?: APIErrorDetail;
}

export interface UserErrorBreakdown {
  user_id: number;
  endpoint: string;
  status_code: number;
  count: number;
  last_seen: string;
}

export interface LatencyTrendPoint {
  period: string;
  avg_ms: number;
  p95_ms: number;
  count: number;
}

export interface SlowestEndpoint {
  endpoint_id: number;
  endpoint: string;
  p95_ms: number;
  avg_ms: number;
  count: number;
}

export interface RecentError {
  endpoint: string;
  status_code: number;
  user_id: number | null;
  duration_ms: number;
  called_at: string;
}

export interface CacheMetrics {
  keyspace_hits: number;
  keyspace_misses: number;
  used_memory_human: string;
  used_memory_peak_human: string;
  instantaneous_ops_per_sec: number;
  raw_stats: Record<string, unknown>;
}

export interface EventBusMetrics {
  EventsPublished: number;
  EventsProcessed: number;
  EventsDropped: number;
  HandlerErrors: number;
  HandlerPanics: number;
  AvgProcessingTime: number;
  QueueDepth: number;
}

export interface TopEndpoint {
  endpoint_id: number;
  method: string;
  path: string;
}

export interface BrowserPlatformStat {
  browser: string;
  platform: string;
  unique_users: number;
  total_actions: number;
}

export interface VersionStat {
  version: string;
  unique_users: number;
  total_actions: number;
}

export interface OSGroupStat {
  os_family: string;
  unique_users: number;
  total_actions: number;
  versions: VersionStat[];
}

export interface BrowserGroupStat {
  browser_family: string;
  unique_users: number;
  total_actions: number;
  versions: VersionStat[];
}

export interface BrowserPlatformUser {
  user_id: number;
  username: string;
  fullname: string;
  role: string;
  actions: number;
  last_seen: string;
}

export interface FailedLoginGeo {
  country: string;
  city: string;
  region: string;
}

export interface FailedLoginAttempt {
  id: number;
  user_id: number;
  user_name: string;
  attempted_identifier: string;
  ip_address: string;
  user_agent: string;
  browser: string;
  platform: string;
  location?: FailedLoginGeo;
  reason: string;
  created_at: string;
}

export interface LoginIdentifierSummary {
  identifier: string;
  count: number;
  last_seen: string;
  reason: string;
}

export interface FailedLoginsResponse {
  attempts: FailedLoginAttempt[];
  summary: LoginIdentifierSummary[];
}

export interface ErrorCountResponse {
  total: number;
}
