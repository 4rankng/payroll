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

  it('explains when a flexible payroll file is uploaded as a timesheet', () => {
    expect(parseImportErrors(JSON.stringify([
      {
        reason: 'tệp này là bảng lương linh hoạt, không phải bảng chấm công BCC',
      },
    ]))).toEqual([
      {
        row: 0,
        employee: '',
        reason: 'Tệp này là bảng lương, không phải bảng chấm công',
      },
    ]);
  });

  it('explains legacy BCC format errors without blaming an employee', () => {
    expect(parseImportErrors(JSON.stringify([
      {
        reason: 'không nhận diện được định dạng file: không nhận diện được định dạng file BCC',
      },
    ]))).toEqual([
      {
        row: 0,
        employee: '',
        reason: 'Tệp không đúng mẫu bảng chấm công',
      },
    ]);
  });
});
