// Payrate key discipline, mirroring the backend (domain/payrate.go
// NormalizedKeys + pkg/utils/text_normalizer.NormalizeVietnamese):
// - Storage/display keys are trimmed and NFC-composed, diacritics preserved —
//   they double as position display names and lookup paths.
// - Duplicate detection folds case, diacritics and Đ/đ away exactly like the
//   Go fold, so "Lái xe nâng" and "lái xe nâng" can never both be persisted.

export function normalizePayrateKey(key: string): string {
  return key.trim().normalize('NFC');
}

export function payrateKeyFold(key: string): string {
  return normalizePayrateKey(key)
    .toLowerCase()
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/g, 'd');
}

// True when no existing key folds to the same canonical form (or is the same
// key being renamed). Used by position/hour-type add+rename guards.
export function isPayrateKeyAvailable(existingKeys: string[], candidate: string, selfKey?: string): boolean {
  const fold = payrateKeyFold(candidate);
  return !existingKeys.some(k => k !== selfKey && payrateKeyFold(k) === fold);
}
