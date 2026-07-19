import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ResponsivePage } from './ResponsivePage';

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: () => true,
}));

describe('ResponsivePage', () => {
  it('keeps the mobile route pinned to the viewport width', () => {
    const { container } = render(
      <ResponsivePage
        desktopComponent={() => <div>Desktop</div>}
        mobileComponent={() => <div>Mobile</div>}
      />,
    );

    expect(screen.getByText('Mobile')).toBeInTheDocument();
    expect(container.querySelector('.mobile-page')).toHaveClass('w-[100cqw]', 'max-w-none');
  });
});
