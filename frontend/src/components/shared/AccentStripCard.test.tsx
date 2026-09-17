import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AccentStripCard } from './AccentStripCard';

describe('card keyboard actions', () => {
  it('opens from the card but does not intercept nested control keys', () => {
    const onOpen = vi.fn();
    render(<AccentStripCard accentColor="green" onClick={onOpen}><span>Nhân viên</span><button>Ứng lương</button></AccentStripCard>);
    fireEvent.keyDown(screen.getByRole('button', { name: 'Nhân viên Ứng lương' }), { key: 'Enter' });
    expect(onOpen).toHaveBeenCalledOnce();
    fireEvent.keyDown(screen.getByRole('button', { name: 'Ứng lương' }), { key: ' ' });
    expect(onOpen).toHaveBeenCalledOnce();
  });
});
