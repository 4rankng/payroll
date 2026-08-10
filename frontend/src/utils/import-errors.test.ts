import { describe, expect, it } from 'vitest';

import { parseImportErrors } from './import-errors';

describe('parseImportErrors', () => {
  it('replaces internal and English details with safe Vietnamese copy', () => {
    const errors = parseImportErrors(JSON.stringify([
      {
        row: 2,
        employee: 'employee_id=91',
        reason: 'OnePay provider failed: timeout employee_id=91',
      },
      {
        row: 3,
        employee: 'Nguyễn Văn An',
        reason: 'không thể tạo nhân viên CCCD 123: database error',
      },
      {
        row: 4,
        employee: 'ID 91',
        reason: 'invalid account info',
      },
    ]));

    expect(errors).toEqual([
      {
        row: 2,
        employee: '',
        reason: 'Không thể xử lý dòng dữ liệu này',
      },
      {
        row: 3,
        employee: 'Nguyễn Văn An',
        reason: 'Không thể tạo hồ sơ nhân viên',
      },
      {
        row: 4,
        employee: '',
        reason: 'Không thể xử lý dòng dữ liệu này',
      },
    ]);
  });

  it('ignores malformed payloads', () => {
    expect(parseImportErrors('{"reason":"failed"}')).toEqual([]);
    expect(parseImportErrors('not-json')).toEqual([]);
  });

  it('keeps a missing-payrate error actionable', () => {
    expect(parseImportErrors(JSON.stringify([{
      row: 14,
      employee: 'Nguyễn Văn Kiên',
      reason: 'không tìm thấy mức lương cho ca HC, vị trí Lương 520, ngày 2026-08-10',
    }]))).toEqual([{
      row: 14,
      employee: 'Nguyễn Văn Kiên',
      reason: 'Chưa cấu hình mức lương phù hợp cho ca làm việc',
    }]);
  });

});
