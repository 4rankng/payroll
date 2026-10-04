import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './tooltip';

describe('tooltip UU surface (W5)', () => {
  it('renders the UU tooltip surface when shown', () => {
    render(
      <TooltipProvider>
        <Tooltip defaultOpen>
          <TooltipTrigger>Tích lũy</TooltipTrigger>
          <TooltipContent>Tổng giờ công tháng này</TooltipContent>
        </Tooltip>
      </TooltipProvider>,
    );
    // Radix mirrors the label into a hidden [role=tooltip] span; the styled
    // surface is that span's parent.
    expect(screen.getByRole('tooltip').parentElement).toHaveClass(
      'bg-popover',
      'border-utility-gray-200',
      'rounded-lg',
      'shadow-lg',
      'typography-body-medium',
    );
  });
});
