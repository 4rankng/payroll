import { validateRejectUnpaidTimesheetsForm } from './useRejectUnpaidTimesheetsDialog';

describe('validateRejectUnpaidTimesheetsForm', () => {
  it('requires one project, a valid inclusive range, and a trimmed reason', () => {
    expect(validateRejectUnpaidTimesheetsForm('', '', '', '   ')).toEqual({
      project: 'Vui lòng chọn một dự án.',
      dateRange: 'Vui lòng chọn đầy đủ khoảng ngày.',
      reason: 'Vui lòng nhập lý do loại.',
    });

    expect(
      validateRejectUnpaidTimesheetsForm('7', '2026-07-31', '2026-07-01', 'Sai dữ liệu'),
    ).toEqual({
      dateRange: 'Ngày bắt đầu không được sau ngày kết thúc.',
    });

    expect(
      validateRejectUnpaidTimesheetsForm('7', '2026-07-01', '2026-07-31', ' Sai dữ liệu '),
    ).toEqual({});
  });
});
