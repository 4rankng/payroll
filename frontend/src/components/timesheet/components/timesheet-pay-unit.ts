import { vietnameseEquals } from "@/utils/vietnameseNormalization";

export function calculateTimesheetPreviewAmount(
  rate: number,
  hoursWorked: number,
  isFlexibleProject: boolean,
): number {
  return isFlexibleProject ? rate : rate * hoursWorked;
}

export function getPayrateUnit(isFlexibleProject: boolean): string {
  return isFlexibleProject ? "đ/ca" : "đ/giờ";
}

export function findTimesheetRate(
  rates: unknown,
  position: string,
  dayType: string,
  hourType: string,
): number | null {
  if (!rates || typeof rates !== "object" || !position) return null;

  const positionEntry = Object.entries(rates as Record<string, unknown>).find(
    ([key]) => vietnameseEquals(key, position),
  );
  if (!positionEntry || !positionEntry[1] || typeof positionEntry[1] !== "object") {
    return null;
  }

  const dayRates = (positionEntry[1] as Record<string, unknown>)[dayType];
  if (!dayRates || typeof dayRates !== "object") return null;

  const rate = (dayRates as Record<string, unknown>)[hourType];
  return typeof rate === "number" ? rate : null;
}
