import { describe, expect, it } from 'vitest';

import {
  describeGroupedError,
  groupImportErrors,
  parseImportErrors,
} from './import-errors';

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

  it('keeps a missing-payrate error actionable with its affected date', () => {
    expect(parseImportErrors(JSON.stringify([{
      row: 14,
      employee: 'Nguyễn Văn Kiên',
      reason: 'không tìm thấy mức lương cho ca HC, vị trí Lương 520, ngày 2026-08-10',
    }]))).toEqual([{
      row: 14,
      employee: 'Nguyễn Văn Kiên',
      reason: 'Chưa cấu hình mức lương phù hợp cho ca làm việc',
      date: '10/08/2026',
    }]);
  });

  it('names the missing shift or position when the payrate config lacks it', () => {
    expect(parseImportErrors(JSON.stringify([
      {
        row: 0,
        employee: '',
        reason: 'ca làm "OT400" không có trong cấu hình lương cho ngày 2026-09-08. Các ca làm khả dụng: OT30, HC, OT150',
      },
      {
        row: 0,
        employee: '',
        reason: 'vị trí "Bảo vệ" không có trong cấu hình lương. Các vị trí khả dụng: HC',
      },
    ]))).toEqual([
      {
        row: 0,
        employee: '',
        reason: 'Ca làm "OT400" chưa có trong bảng lương của dự án',
        date: '08/09/2026',
      },
      {
        row: 0,
        employee: '',
        reason: 'Vị trí "Bảo vệ" chưa có trong bảng lương của dự án',
      },
    ]);
  });

  it('falls back to the payrate hint when the config error names no safe shift', () => {
    expect(parseImportErrors(JSON.stringify([{
      row: 0,
      employee: '',
      reason: 'không tìm thấy cấu hình lương cho ngày 2026-09-08',
    }]))).toEqual([{
      row: 0,
      employee: '',
      reason: 'Chưa cấu hình bảng lương cho dự án',
      date: '08/09/2026',
    }]);
  });

  it('maps a missing project payrate to a config hint instead of the generic fallback', () => {
    expect(parseImportErrors(JSON.stringify([{
      row: 0,
      employee: '',
      reason: 'không tìm thấy bảng lương cho dự án: no active payrate found for this project and date',
    }]))).toEqual([{
      row: 0,
      employee: '',
      reason: 'Chưa cấu hình bảng lương cho dự án',
    }]);
  });

  it('passes the out-of-month file-level reason through with its data range intact', () => {
    const raw =
      'file chỉ chứa giờ công ngoài tháng 2026-09 (dữ liệu từ 22/08/2026 đến 28/08/2026) — chọn tháng tương ứng với dữ liệu file';
    expect(parseImportErrors(JSON.stringify([{ row: 0, employee: '', reason: raw }])))
      .toEqual([{ row: 0, employee: '', reason: raw }]);
  });

  it('keeps the affected date beside the future-attendance reason so duplicates stay groupable', () => {
    expect(parseImportErrors(JSON.stringify([{
      employee: 'Nguyễn Văn An',
      reason: 'ngày 2026-08-16: Ngày chấm công chưa đến',
    }]))).toEqual([{
      row: 0,
      employee: 'Nguyễn Văn An',
      reason: 'Ngày chấm công chưa đến',
      date: '16/08/2026',
    }]);
  });

  it('maps paid and uncreatable timesheet reasons to distinct actionable copy', () => {
    expect(parseImportErrors(JSON.stringify([
      { employee: 'Trần B', reason: 'ngày 2026-08-20: Bảng chấm công đã thanh toán' },
      { employee: 'Trần C', reason: 'ngày 2026-08-21: Không thể tạo bảng chấm công' },
    ]))).toEqual([
      { row: 0, employee: 'Trần B', reason: 'Bảng chấm công đã thanh toán', date: '20/08/2026' },
      { row: 0, employee: 'Trần C', reason: 'Không thể tạo bảng chấm công', date: '21/08/2026' },
    ]);
  });
});

describe('groupImportErrors', () => {
  it('collapses identical employee+reason rows into one group with their dates', () => {
    const errors = parseImportErrors(JSON.stringify([
      { employee: 'Đặng Mai Lan', reason: 'ngày 2026-06-24: Bảng chấm công đã được phê duyệt' },
      { employee: 'Đặng Mai Lan', reason: 'ngày 2026-06-25: Bảng chấm công đã được phê duyệt' },
      { employee: 'Đặng Mai Lan', reason: 'ngày 2026-06-26: Bảng chấm công đã được phê duyệt' },
      { employee: 'Lương Thị Tứ', reason: 'ngày 2026-06-25: Bảng chấm công đã được phê duyệt' },
      { employee: 'Đặng Mai Lan', reason: 'ngày 2026-06-25: Ngày chấm công chưa đến' },
    ]));

    expect(groupImportErrors(errors)).toEqual([
      {
        employee: 'Đặng Mai Lan',
        reason: 'Bảng chấm công đã được phê duyệt',
        count: 3,
        dates: ['24/06/2026', '25/06/2026', '26/06/2026'],
        rows: [],
      },
      {
        employee: 'Lương Thị Tứ',
        reason: 'Bảng chấm công đã được phê duyệt',
        count: 1,
        dates: ['25/06/2026'],
        rows: [],
      },
      {
        employee: 'Đặng Mai Lan',
        reason: 'Ngày chấm công chưa đến',
        count: 1,
        dates: ['25/06/2026'],
        rows: [],
      },
    ]);
  });

  it('falls back to row numbers when no dates are available', () => {
    const errors = parseImportErrors(JSON.stringify([
      { row: 12, employee: 'Ngô Văn D', reason: 'nhân viên không tìm thấy trong hệ thống' },
      { row: 13, employee: 'Ngô Văn D', reason: 'nhân viên không tìm thấy trong hệ thống' },
    ]));

    expect(groupImportErrors(errors)).toEqual([
      {
        employee: 'Ngô Văn D',
        reason: 'Không tìm thấy nhân viên',
        count: 2,
        dates: [],
        rows: [12, 13],
      },
    ]);
  });
});

describe('describeGroupedError', () => {
  it('shows a single date plainly', () => {
    expect(
      describeGroupedError({
        employee: 'An',
        reason: 'R',
        count: 1,
        dates: ['25/06/2026'],
        rows: [],
      }),
    ).toBe(' (ngày 25/06/2026)');
  });

  it('lists several dates and caps long lists with the row count', () => {
    const dates = Array.from({ length: 26 }, (_, i) => `${String(i + 1).padStart(2, '0')}/07/2026`);
    expect(
      describeGroupedError({ employee: 'An', reason: 'R', count: 265, dates, rows: [] }),
    ).toBe(' (các ngày 01/07/2026, 02/07/2026, 03/07/2026, 04/07/2026, 05/07/2026…, 265 dòng)');
  });

  it('falls back to row numbers, then to a bare count', () => {
    expect(
      describeGroupedError({ employee: 'An', reason: 'R', count: 2, dates: [], rows: [12] }),
    ).toBe(' (dòng 12)');
    expect(
      describeGroupedError({ employee: '', reason: 'R', count: 4, dates: [], rows: [] }),
    ).toBe(' (4 dòng)');
    expect(
      describeGroupedError({ employee: 'An', reason: 'R', count: 1, dates: [], rows: [] }),
    ).toBe('');
  });
});
