import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import NotFound from './NotFound';

const auth = vi.hoisted(() => ({ role: 'employee', valid: true }));
vi.mock('@/lib/auth', () => ({ authManager: { isTokenValid: () => auth.valid, getUserRole: () => auth.role } }));
vi.mock('@/lib/logger', () => ({ logNotFoundPath: vi.fn() }));

describe('404 recovery', () => {
  it.each(['admin', 'partner', 'adv_partner', 'employee', 'accountant'])('returns %s to the role-aware entry point', (role) => {
    auth.role = role;
    auth.valid = true;
    render(<MemoryRouter initialEntries={['/missing']}><Routes><Route path="/missing" element={<NotFound />} /><Route path="/" element={<p>Role home</p>} /></Routes></MemoryRouter>);
    fireEvent.click(screen.getByRole('button', { name: 'Về bảng điều khiển' }));
    expect(screen.getByText('Role home')).toBeInTheDocument();
  });
});
