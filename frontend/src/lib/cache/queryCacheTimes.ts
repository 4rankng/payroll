/**
 * Shared query cache timings.
 *
 * Reference data (projects, banks, settings) changes rarely and every mutation
 * invalidates its keys on success, so these queries tolerate a longer
 * staleTime than the global 30s default — this skips redundant
 * focus/interval refetches of unchanged data without hiding changes.
 */
export const REFERENCE_DATA_STALE_TIME_MS = 5 * 60 * 1000;
