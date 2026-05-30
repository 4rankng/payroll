export interface CronJob {
  name: string;
  cron: string;
  is_enabled: boolean;
  last_status: 'success' | 'failed' | 'running' | null;
  last_run: string | null;
  last_duration_ms: number | null;
  last_error: string | null;
}
