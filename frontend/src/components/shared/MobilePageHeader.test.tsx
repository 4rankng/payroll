import { render, screen } from '@testing-library/react';
import { MobilePageHeader } from './MobilePageHeader';

describe('MobilePageHeader', () => {
  it('gives actions a full-width row below the title on narrow screens', () => {
    render(
      <MobilePageHeader
        title="Bảng công"
        subtitle="Theo dõi và duyệt bảng công"
        actions={<button type="button">Nhập</button>}
      />,
    );

    const action = screen.getByRole('button', { name: 'Nhập' });
    const actionsContainer = action.parentElement;
    const headerRow = screen.getByRole('heading', { name: 'Bảng công' }).parentElement?.parentElement?.parentElement;

    expect(actionsContainer).toHaveClass('basis-full', 'max-w-full', 'sm:basis-auto');
    expect(headerRow).toHaveClass('flex-wrap', 'sm:flex-nowrap');
  });

  it('uses the page canvas when rendered as an embedded borderless header', () => {
    render(
      <MobilePageHeader
        title="Bảng công"
        sticky={false}
        bordered={false}
      />,
    );

    const headerRow = screen.getByRole('heading', { name: 'Bảng công' }).parentElement?.parentElement?.parentElement;
    const header = headerRow?.parentElement;

    expect(header).toHaveClass('bg-transparent');
    expect(header).not.toHaveClass('bg-white');
  });

  it('keeps a short action beside the title when inline layout is requested', () => {
    render(
      <MobilePageHeader
        title="Dự án"
        actionsLayout="inline"
        actions={<button type="button">Tạo dự án</button>}
      />,
    );

    const action = screen.getByRole('button', { name: 'Tạo dự án' });
    const actionsContainer = action.parentElement;
    const headerRow = screen.getByRole('heading', { name: 'Dự án' }).parentElement?.parentElement?.parentElement;

    expect(actionsContainer).toHaveClass('basis-auto', 'max-w-[52%]');
    expect(headerRow).toHaveClass('flex-nowrap');
    expect(headerRow).not.toHaveClass('flex-wrap');
  });
});
