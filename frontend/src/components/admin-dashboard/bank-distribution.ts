import type { BankUsageItem } from '@/types/api/dashboard.types';

export interface BankDistributionItem {
  name: string;
  shortName: string;
  employeeCount: number;
  percentage: number;
  color: string;
  transferCount: number;
  totalPaidVnd: number;
}

const BANK_COLORS = [
  '#2563eb',
  '#0284c7',
  '#059669',
  '#d97706',
  '#e11d48',
] as const;
const OTHER_COLOR = '#64748b';
const MAX_NAMED_BANKS = 5;

function extractShortName(bankName: string): string {
  const match = bankName.match(/\(([^)]+)\)/);
  return match ? match[1] : bankName.slice(0, 3).toUpperCase();
}

function calculatePercentage(employeeCount: number, totalEmployees: number): number {
  return totalEmployees > 0 ? Math.round((employeeCount / totalEmployees) * 100) : 0;
}

export function buildBankDistribution(
  banks: BankUsageItem[],
  totalEmployees: number,
): BankDistributionItem[] {
  if (banks.length === 0) {
    return [];
  }

  const sortedBanks = [...banks].sort((left, right) => right.employee_count - left.employee_count);
  const namedBanks = sortedBanks.slice(0, MAX_NAMED_BANKS);
  const remainingBanks = sortedBanks.slice(MAX_NAMED_BANKS);

  const distribution = namedBanks.map((bank, index) => ({
    name: bank.bank_name,
    shortName: extractShortName(bank.bank_name),
    employeeCount: bank.employee_count,
    percentage: calculatePercentage(bank.employee_count, totalEmployees),
    color: BANK_COLORS[index] ?? OTHER_COLOR,
    transferCount: bank.transfer_count,
    totalPaidVnd: bank.total_paid_vnd,
  }));

  if (remainingBanks.length > 0) {
    const employeeCount = remainingBanks.reduce(
      (total, bank) => total + bank.employee_count,
      0,
    );

    distribution.push({
      name: 'Ngân hàng khác',
      shortName: 'Khác',
      employeeCount,
      percentage: calculatePercentage(employeeCount, totalEmployees),
      color: OTHER_COLOR,
      transferCount: remainingBanks.reduce(
        (total, bank) => total + bank.transfer_count,
        0,
      ),
      totalPaidVnd: remainingBanks.reduce(
        (total, bank) => total + bank.total_paid_vnd,
        0,
      ),
    });
  }

  return distribution;
}
