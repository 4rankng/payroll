// Local fee-preview calculator. Mirrors the backend resolver in
// internal/domain/advance_payment_fee_schedule.go (ResolveFee + tierFor) so
// the form's "Phí cho khoản 5,000,000 VND" preview can update without a
// round-trip while the user is still editing.

import { useMemo } from "react";

import type { FeeScheduleTier } from "@/types/api/advance-payment-fee-schedule.types";

export interface FeePreviewInput {
  amount: number;
  tiers: FeeScheduleTier[];
  minFeeVnd: number;
}

export interface FeePreviewResult {
  fee: number;
  netAmount: number;
  appliedTier: FeeScheduleTier | null; // null = config invalid / no tiers
}

/**
 * Picks the highest-MinAmount tier whose minAmount <= amount. Tiers must be
 * sorted ascending by minAmount; the form keeps them sorted on save.
 */
export function resolveFeeLocal(input: FeePreviewInput): FeePreviewResult {
  const { amount, tiers, minFeeVnd } = input;
  if (!Number.isFinite(amount) || amount <= 0 || tiers.length === 0) {
    return { fee: 0, netAmount: 0, appliedTier: null };
  }

  let applied = tiers[0];
  for (const tier of tiers) {
    if (tier.minAmount <= amount) {
      applied = tier;
    } else {
      break;
    }
  }

  const percentageFee = Math.floor((amount * applied.percentage) / 100);
  const fee = Math.max(percentageFee, minFeeVnd);
  const cappedFee = fee >= amount ? amount : fee;
  return { fee: cappedFee, netAmount: amount - cappedFee, appliedTier: applied };
}

export function useFeeSchedulePreview(input: FeePreviewInput): FeePreviewResult {
  return useMemo(() => resolveFeeLocal(input), [input]);
}
