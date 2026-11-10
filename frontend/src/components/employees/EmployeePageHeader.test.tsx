import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { EmployeePageHeader } from './EmployeePageHeader';

const mockUser = { role: 'admin' };

vi.mock('@/contexts', () => ({
  useAuth: () => ({ user: mockUser }),
}));

const renderHeader = (props: Partial<Parameters<typeof EmployeePageHeader>[0]> = {}) =>
  render(
    <MemoryRouter>
      <EmployeePageHeader
        totalEmployees={1632}
        onAddEmployeeClick={() => {}}
        onExportClick={() => {}}
        onExportNoMobileClick={() => {}}
        onImportMobilesClick={() => {}}
        {...props}
      />
    </MemoryRouter>,
  );

describe('EmployeePageHeader', () => {
  it('renders admin actions as one segmented toolbar with the primary action last', () => {
    mockUser.role = 'admin';
    renderHeader();

    const toolbar = screen.getByTestId('employee-header-toolbar');
    // One bordered unit on the card token — no loose wrapping buttons.
    expect(toolbar).toHaveClass('rounded-xl', 'border', 'border-border', 'bg-card');

    const segments = within(toolbar).getAllByRole('button');
    expect(segments.map((segment) => segment.textContent)).toEqual([
      'Xuất Excel',
      'NV chưa có SĐT',
      'Nhập SĐT',
      'Thêm nhân viên',
    ]);
    // Hairline separators between segments only — none after the final CTA.
    expect(segments[0]).toHaveClass('border-r');
    expect(segments[segments.length - 1]).not.toHaveClass('border-r');
    // The primary CTA ends the strip.
    expect(segments[segments.length - 1]).toHaveClass('bg-primary');
  });

  it('hides admin-only export segments for partners', () => {
    mockUser.role = 'partner';
    renderHeader();

    const toolbar = screen.getByTestId('employee-header-toolbar');
    const labels = within(toolbar).getAllByRole('button').map((segment) => segment.textContent);
    expect(labels).toEqual(['Xuất Excel', 'Thêm nhân viên']);
  });

  it('marks export segments inert while an export runs and keeps the CTA live', () => {
    mockUser.role = 'admin';
    renderHeader({ isExporting: true });

    const toolbar = screen.getByTestId('employee-header-toolbar');
    const segments = within(toolbar).getAllByRole('button');

    expect(segments[0]).toHaveTextContent('Đang xuất...');
    expect(segments[1]).toHaveTextContent('Đang xuất...');
    for (const exporting of segments.slice(0, 2)) {
      expect(exporting).toHaveClass('pointer-events-none', 'opacity-50');
      expect(exporting).toHaveAttribute('aria-busy', 'true');
    }
    expect(segments[segments.length - 1]).not.toHaveClass('pointer-events-none');
    expect(segments[segments.length - 1]).not.toHaveAttribute('aria-busy');
  });
});
