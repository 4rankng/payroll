import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { MetadataRenderer } from './MetadataRenderer';

describe('MetadataRenderer', () => {
  it('shows backend before and after values for changed fields', () => {
    const metadata = JSON.stringify({
      changed_fields: {
        date_of_birth: {
          before: '1990-01-02',
          after: '1991-03-04',
        },
        bank_account_number: {
          before: '0123456789',
          after: '9876543210',
        },
      },
    });

    render(<MetadataRenderer metadata={metadata} />);

    expect(screen.getByText('1990-01-02')).toBeInTheDocument();
    expect(screen.getByText('1991-03-04')).toBeInTheDocument();
    expect(screen.getByText('0123456789')).toBeInTheDocument();
    expect(screen.getByText('9876543210')).toBeInTheDocument();
  });

  it('keeps rendering legacy old and new change values', () => {
    const metadata = JSON.stringify({
      changed_fields: {
        fullname: {
          old: 'Nguyễn Văn A',
          new: 'Nguyễn Văn B',
        },
      },
    });

    render(<MetadataRenderer metadata={metadata} />);

    expect(screen.getByText('Nguyễn Văn A')).toBeInTheDocument();
    expect(screen.getByText('Nguyễn Văn B')).toBeInTheDocument();
  });

  it('renders flat create metadata as a readable created-value list', () => {
    const metadata = JSON.stringify({
      name: 'Dự án Riverside',
      code: 'RS-01',
      client_name: '',
    });

    render(<MetadataRenderer metadata={metadata} action="CREATE" entityType="project" />);

    expect(screen.getByText('Giá trị được tạo')).toBeInTheDocument();
    expect(screen.getByText('Tên')).toBeInTheDocument();
    expect(screen.getByText('Dự án Riverside')).toBeInTheDocument();
    expect(screen.getByText('Trống')).toBeInTheDocument();
    expect(screen.queryByText(/\{"name"/)).not.toBeInTheDocument();
  });

  it('does not classify a business reason as authentication metadata', () => {
    const metadata = JSON.stringify({
      project_name: 'Dự án Riverside',
      employee_name: 'Nguyễn Văn A',
      reason: 'Kết thúc phân công',
    });

    render(<MetadataRenderer metadata={metadata} action="DELETE" entityType="project_employee" />);

    expect(screen.getByText('Giá trị đã xóa')).toBeInTheDocument();
    expect(screen.queryByText('Chi tiết xác thực')).not.toBeInTheDocument();
    expect(screen.getByText('Kết thúc phân công')).toBeInTheDocument();
  });

  it('normalizes paired old and new fields into a diff', () => {
    const metadata = JSON.stringify({
      old_fullname: 'Nguyễn Văn A',
      new_fullname: 'Nguyễn Văn B',
      cccd: '012345678901',
    });

    render(<MetadataRenderer metadata={metadata} action="UPDATE" entityType="employee" />);

    expect(screen.getByText('Thay đổi')).toBeInTheDocument();
    expect(screen.getByText('Nguyễn Văn A')).toBeInTheDocument();
    expect(screen.getByText('Nguyễn Văn B')).toBeInTheDocument();
    expect(screen.getByText('CCCD')).toBeInTheDocument();
  });

  it('shows invalid metadata instead of silently claiming there are no details', () => {
    render(<MetadataRenderer metadata="{invalid" action="UPDATE" entityType="employee" />);

    expect(screen.getByText('Dữ liệu chi tiết không hợp lệ')).toBeInTheDocument();
  });

  it('rejects mixed valid and malformed field changes instead of hiding part of the audit', () => {
    const metadata = JSON.stringify({
      changed_fields: {
        fullname: { before: 'Nguyễn Văn A', after: 'Nguyễn Văn B' },
        mobile: { before: '0900000000' },
      },
    });

    render(<MetadataRenderer metadata={metadata} action="UPDATE" entityType="employee" />);

    expect(screen.getByText('Dữ liệu chi tiết không hợp lệ')).toBeInTheDocument();
    expect(screen.queryByText('Nguyễn Văn B')).not.toBeInTheDocument();
  });
});
