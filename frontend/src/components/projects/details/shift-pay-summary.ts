const SHIFT_RANGE_RE =
  /^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$/;

export interface ShiftPositionPay {
  position: string;
  amount: number;
}

export interface ShiftPaySummary {
  range: string;
  positionPays: ShiftPositionPay[];
}

/**
 * Reads flexible-project payrates without transforming their unit.
 * Every configured value is the total wage for one completed shift.
 */
export function extractShiftPaySummaries(rates: unknown): ShiftPaySummary[] {
  if (!rates || typeof rates !== "object") return [];

  const summaries = new Map<string, Map<string, number>>();
  for (const [position, dayTypes] of Object.entries(
    rates as Record<string, unknown>,
  )) {
    if (!dayTypes || typeof dayTypes !== "object") continue;

    for (const hourRanges of Object.values(
      dayTypes as Record<string, unknown>,
    )) {
      if (!hourRanges || typeof hourRanges !== "object") continue;

      for (const [range, rawAmount] of Object.entries(
        hourRanges as Record<string, unknown>,
      )) {
        if (!SHIFT_RANGE_RE.test(range) || typeof rawAmount !== "number") {
          continue;
        }

        const positionPays =
          summaries.get(range) ?? new Map<string, number>();
        // Flexible configs normally have one "ngày thường" entry. Keeping the
        // first value avoids duplicating a position in legacy multi-day data.
        if (!positionPays.has(position)) {
          positionPays.set(position, rawAmount);
        }
        summaries.set(range, positionPays);
      }
    }
  }

  return Array.from(summaries, ([range, positionPays]) => ({
    range,
    positionPays: Array.from(positionPays, ([position, amount]) => ({
      position,
      amount,
    })).sort((a, b) => a.position.localeCompare(b.position, "vi")),
  })).sort((a, b) =>
    a.range.slice(0, 5).localeCompare(b.range.slice(0, 5)),
  );
}

export function formatShiftPay(amount: number): string {
  return `${amount.toLocaleString("vi-VN")} đ`;
}
