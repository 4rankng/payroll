/**
 * Track recent bulk-transfer exports locally so we can hint the user that
 * they have an Excel file waiting to be uploaded back as a result.
 *
 * Note: this is a UX hint only. The backend currently does NOT persist
 * "exports pending upload" — each export is independent of any upload.
 * If/when the backend gains a `bulk_transfers` table tracking export→upload
 * lifecycle, this utility can be replaced with a server query.
 */

const STORAGE_KEY = 'pendingBulkTransferExports';
const MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000; // 7 days
const MAX_ENTRIES = 5;

export interface PendingExport {
  id: string;            // local uuid
  exportedAt: string;    // ISO datetime
  fromDate?: string;     // YYYY-MM-DD
  toDate?: string;       // YYYY-MM-DD
  forMonth?: string;     // YYYY-MM
  filename?: string;     // best-effort, may be undefined if browser blocked it
  employeeIdsCount?: number;
  projectIdsCount?: number;
}

function readAll(): PendingExport[] {
  if (typeof window === 'undefined' || !window.localStorage) return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const list = JSON.parse(raw) as PendingExport[];
    if (!Array.isArray(list)) return [];
    // Drop expired entries
    const now = Date.now();
    return list.filter((e) => {
      const t = Date.parse(e.exportedAt);
      return Number.isFinite(t) && now - t < MAX_AGE_MS;
    });
  } catch {
    return [];
  }
}

function writeAll(list: PendingExport[]): void {
  if (typeof window === 'undefined' || !window.localStorage) return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(list.slice(0, MAX_ENTRIES)));
  } catch {
    // ignore quota errors
  }
}

export function getPendingExports(): PendingExport[] {
  return readAll();
}

export function getMostRecentPendingExport(): PendingExport | null {
  const list = readAll();
  if (list.length === 0) return null;
  return list.reduce<PendingExport | null>((best, cur) => {
    if (!best) return cur;
    return Date.parse(cur.exportedAt) > Date.parse(best.exportedAt) ? cur : best;
  }, null);
}

export function recordPendingExport(input: Omit<PendingExport, 'id' | 'exportedAt'>): PendingExport {
  const entry: PendingExport = {
    id: `pe_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`,
    exportedAt: new Date().toISOString(),
    ...input,
  };
  const list = readAll();
  list.unshift(entry);
  writeAll(list);
  return entry;
}

export function dismissPendingExport(id: string): void {
  const list = readAll().filter((e) => e.id !== id);
  writeAll(list);
}

export function clearAllPendingExports(): void {
  writeAll([]);
}
