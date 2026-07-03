export interface FailedAttemptOverrideViewState {
  canOverride: boolean;
  isResolved: boolean;
  actionLabel: string;
  dialogTitle: string;
  resolvedLabel: string;
}

export function isGpsDeviceFailure(reasonCategory: string): boolean {
  return reasonCategory.startsWith("gps_");
}

export function getFailedAttemptOverrideViewState(
  reasonCategory: string,
  resolvedAt?: string | null
): FailedAttemptOverrideViewState {
  const canOverride = isGpsDeviceFailure(reasonCategory);
  const isResolved = Boolean(resolvedAt);

  return {
    canOverride,
    isResolved,
    actionLabel: "Cần duyệt GPS",
    dialogTitle: "Duyệt ghi nhận chấm công",
    resolvedLabel: "Đã ghi nhận",
  };
}
