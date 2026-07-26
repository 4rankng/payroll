import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ResponsivePage } from './ResponsivePage';

const { useIsMobileMock } = vi.hoisted(() => ({
  useIsMobileMock: vi.fn(),
}));

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: useIsMobileMock,
}));

describe('ResponsivePage', () => {
  beforeEach(() => {
    useIsMobileMock.mockReset();
  });

  it('keeps the mobile route pinned to the viewport width', () => {
    useIsMobileMock.mockReturnValue(true);

    const { container } = render(
      <ResponsivePage
        desktopComponent={() => <div>Desktop</div>}
        mobileComponent={() => <div>Mobile</div>}
      />,
    );

    expect(screen.getByText('Mobile')).toBeInTheDocument();
    expect(container.querySelector('.mobile-page')).toHaveClass('w-[100cqw]', 'max-w-none');
  });

  it('renders the desktop component without the mobile wrapper', () => {
    useIsMobileMock.mockReturnValue(false);

    const { container } = render(
      <ResponsivePage
        desktopComponent={() => <div>Desktop</div>}
        mobileComponent={() => <div>Mobile</div>}
      />,
    );

    expect(screen.getByText('Desktop')).toBeInTheDocument();
    expect(screen.queryByText('Mobile')).not.toBeInTheDocument();
    expect(container.querySelector('.mobile-page')).not.toBeInTheDocument();
  });
});
