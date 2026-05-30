import { format } from "date-fns";
export const HTTP_STATUS: Record<number, string> = {
  400: "Bad Request",
  401: "Unauthorized",
  403: "Forbidden",
  404: "Not Found",
  422: "Unprocessable",
  429: "Too Many Requests",
  500: "Internal Server Error",
  502: "Bad Gateway",
  503: "Service Unavailable",
  504: "Gateway Timeout",
};

export const ROLE_LABELS: Record<string, string> = {
  admin: "Admin",
  partner: "Quản Lý",
  employee: "Nhân Viên",
};

export const METHOD_COLORS: Record<string, string> = {
  GET:    "bg-sky-100 text-sky-700 dark:bg-sky-900/50 dark:text-sky-300",
  POST:   "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300",
  PUT:    "bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300",
  PATCH:  "bg-orange-100 text-orange-700 dark:bg-orange-900/50 dark:text-orange-300",
  DELETE: "bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300",
};

export function roleLabel(r: string) {
  return ROLE_LABELS[r] ?? r;
}

export function statusLabel(code: number) {
  return HTTP_STATUS[code] ? `${code} ${HTTP_STATUS[code]}` : `${code}`;
}

export function relativeTime(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const m = Math.floor(diff / 60000);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (m < 1) return "vừa xong";
  if (m < 60) return `${m}p`;
  if (h < 24) return `${h}h`;
  if (d < 7) return `${d}d`;
  return format(new Date(dateStr), 'dd/MM/yyyy');
}

export function parseEndpoint(endpoint: string): { method: string; path: string } | null {
  const m = endpoint.match(/^(GET|POST|PUT|PATCH|DELETE)\s+(.+)$/);
  if (!m) return null;
  return { method: m[1], path: m[2] };
}

export type PillStatus = "ok" | "warn" | "danger";

export function errorRateStatus(rate: number): PillStatus {
  return rate > 5 ? "danger" : rate > 1 ? "warn" : "ok";
}

export function latencyStatus(ms: number): PillStatus {
  return ms > 2000 ? "danger" : ms > 1000 ? "warn" : "ok";
}
