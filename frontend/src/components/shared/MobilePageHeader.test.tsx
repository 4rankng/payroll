import { render, screen } from '@testing-library/react';
import { MobilePageHeader } from './MobilePageHeader';

describe('MobilePageHeader', () => {
  it('keeps a short action beside the title by default on mobile', () => {
    render(
      <MobilePageHeader
        title="Bảng công"
        subtitle="Theo dõi và duyệt bảng công"
        actions={<button type="button">Nhập</button>}
      />,
    );

    const action = screen.getByRole('button', { name: 'Nhập' });
    const actionsContainer = action.parentElement;
    const titleBlock = screen.getByRole('heading', { name: 'Bảng công' }).parentElement;
    const leftBlock = titleBlock?.parentElement;
    const headerRow = leftBlock?.parentElement;

    // Inline actions keep their beside-the-title sizing, and the title block
    // reserves a 10rem floor so the row only wraps when a narrow viewport
    // genuinely cannot fit both.
    expect(actionsContainer).toHaveClass('basis-auto', 'max-w-[52%]');
    expect(leftBlock).toHaveClass('min-w-[10rem]');
    expect(headerRow).toHaveClass('flex-wrap');
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

  it('stacks actions onto their own row when a page requests stacked layout', () => {
    render(
      <MobilePageHeader
        title="Dự án"
        actionsLayout="stacked"
        actions={<button type="button">Tạo dự án</button>}
      />,
    );

    const action = screen.getByRole('button', { name: 'Tạo dự án' });
    const actionsContainer = action.parentElement;
    const headerRow = screen.getByRole('heading', { name: 'Dự án' }).parentElement?.parentElement?.parentElement;

    expect(actionsContainer).toHaveClass('basis-full', 'max-w-full', 'sm:basis-auto');
    expect(headerRow).toHaveClass('flex-wrap', 'sm:flex-nowrap');
  });
});
