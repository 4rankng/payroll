interface SettlementDateRecord {
  settlement_date: string;
}

const BUSINESS_DATE_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;

function isValidBusinessDate(value: string): boolean {
  const match = BUSINESS_DATE_PATTERN.exec(value);
  if (!match) {
    return false;
  }

  const [, year, month, day] = match;
  const parsedDate = new Date(Date.UTC(Number(year), Number(month) - 1, Number(day)));

  return parsedDate.getUTCFullYear() === Number(year)
    && parsedDate.getUTCMonth() === Number(month) - 1
    && parsedDate.getUTCDate() === Number(day);
}

export function getLatestSettlementDate(
  settlements?: readonly SettlementDateRecord[],
): string | null {
  const validDates = settlements
    ?.map((settlement) => settlement.settlement_date)
    .filter(isValidBusinessDate);

  if (!validDates?.length) {
    return null;
  }

  return validDates.reduce((latestDate, settlementDate) => (
    settlementDate > latestDate ? settlementDate : latestDate
  ));
}

export function formatSettlementBusinessDate(settlementDate: string): string {
  const match = BUSINESS_DATE_PATTERN.exec(settlementDate);
  if (!match || !isValidBusinessDate(settlementDate)) {
    return settlementDate;
  }

  const [, year, month, day] = match;
  return `${day} tháng ${month} ${year}`;
}
