import { describe, expect, it } from 'vitest';

import { getBankInformationWarningReason } from './bank-information-warning';

const completeBankInformation = {
  bank: { id: 1 },
  bank_account_number: '123456789',
  bank_account_name: 'NGUYEN VAN AN',
  bank_account_status: 'valid' as const,
  bank_account_invalid_reason: null,
};

describe('getBankInformationWarningReason', () => {
  it.each([
    [{ ...completeBankInformation, bank: null }, 'Thiếu ngân hàng'],
    [{ ...completeBankInformation, bank_account_number: ' ' }, 'Thiếu số tài khoản'],
    [{ ...completeBankInformation, bank_account_name: '' }, 'Thiếu tên chủ tài khoản'],
  ])('returns the precise missing-field reason', (employee, expected) => {
    expect(getBankInformationWarningReason(employee)).toBe(expected);
  });

  it('keeps a recognized, sanitized Vietnamese invalid reason', () => {
    expect(
      getBankInformationWarningReason({
        ...completeBankInformation,
        bank_account_status: 'invalid',
        bank_account_invalid_reason:
          'Tên chủ tài khoản không khớp với ngân hàng (ngân hàng ghi: TRAN THI B)',
      }),
    ).toBe(
      'Tên chủ tài khoản không khớp với ngân hàng (ngân hàng ghi: TRAN THI B)',
    );
  });

  it.each([
    'OnePay account_not_found',
    'Invalid account info',
    'Lỗi nội bộ khi gọi provider',
    'Thông báo tùy ý chưa được duyệt',
    null,
  ])('replaces untrusted invalid reason %s with safe generic copy', (reason) => {
    expect(
      getBankInformationWarningReason({
        ...completeBankInformation,
        bank_account_status: 'invalid',
        bank_account_invalid_reason: reason,
      }),
    ).toBe('Tài khoản ngân hàng không hợp lệ');
  });
});
