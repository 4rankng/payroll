import { render } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';
import { Badge } from './badge';

describe('Badge contract', () => {
  it('renders every variant with its UU/legacy signature class', () => {
    const signatures: Record<string, string> = {
      default: 'bg-utility-brand-50',
      secondary: 'bg-utility-gray-50',
      destructive: 'bg-utility-error-50',
      outline: 'ring-utility-gray-300',
      success: 'bg-utility-success-50',
      warning: 'bg-utility-warning-50',
      info: 'bg-sky-600',
      admin: 'bg-blue-600',
      partner: 'bg-brand-solid',
      manager: 'bg-teal-600',
      role: 'bg-utility-gray-600',
    };
    for (const [variant, signature] of Object.entries(signatures)) {
      const { getByText, unmount } = render(<Badge variant={variant as never}>Nội dung</Badge>);
      const el = getByText('Nội dung');
      expect(el).toHaveClass(signature);
      expect(el.className).toContain('ring-1'); // UU inset ring base everywhere
      unmount();
    }
  });

  it('lets the call-site className win the merge', () => {
    render(<Badge className="rounded-full">Nội dung</Badge>);
    const el = document.querySelector('span');
    expect(el?.className).toContain('rounded-full');
    expect(el?.className).not.toContain('rounded-md');
  });

  it('passes the ref through to the span', () => {
    const ref = createRef<HTMLSpanElement>();
    render(<Badge ref={ref}>Nội dung</Badge>);
    expect(ref.current).toBeInstanceOf(HTMLSpanElement);
  });
});
