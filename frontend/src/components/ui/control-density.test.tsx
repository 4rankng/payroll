import { render, screen } from '@testing-library/react';
import { readFileSync } from 'node:fs';
import { describe, expect, it, vi } from 'vitest';

import { FilterPill } from '@/components/shared/FilterPill';
import { SearchBar } from '@/components/shared/SearchBar';
import { EmptyState } from '@/components/shared/EmptyState';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

describe('shared control density', () => {
  it('keeps 44px mobile targets while compacting desktop controls', () => {
    render(
      <>
        <Button>Nút chung</Button>
        <Input aria-label="Ô nhập chung" />
        <Select defaultValue="all">
          <SelectTrigger aria-label="Bộ chọn chung">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Tất cả</SelectItem>
          </SelectContent>
        </Select>
        <SearchBar searchTerm="" onSearchChange={vi.fn()} placeholder="Tìm kiếm chung" />
        <FilterPill
          value="all"
          onChange={vi.fn()}
          placeholder="Lọc chung"
          options={[{ value: 'active', label: 'Đang hoạt động' }]}
        />
      </>,
    );

    expect(screen.getByRole('button', { name: 'Nút chung' })).toHaveClass('h-11', 'sm:h-9', 'text-sm');
    expect(screen.getByLabelText('Ô nhập chung')).toHaveClass('h-11', 'sm:h-9');
    expect(screen.getByRole('combobox', { name: 'Bộ chọn chung' })).toHaveClass('h-11', 'sm:h-9');
    expect(screen.getByRole('textbox', { name: 'Tìm kiếm chung' }).parentElement).toHaveClass(
      'min-h-11',
      'sm:min-h-9',
    );
    expect(screen.getByRole('combobox', { name: 'Lọc chung' })).toHaveClass(
      'min-h-11',
      'sm:min-h-9',
    );
  });

  it('uses compact card padding at desktop widths', () => {
    render(
      <Card>
        <CardHeader data-testid="card-header">Tiêu đề</CardHeader>
        <CardContent data-testid="card-content">Nội dung</CardContent>
      </Card>,
    );

    expect(screen.getByTestId('card-header')).toHaveClass('p-3', 'sm:p-4');
    expect(screen.getByTestId('card-content')).toHaveClass('p-3', 'sm:p-4', 'pt-0');
  });

  it('keeps ordinary application copy on the 11px and 12px density scale', () => {
    const variables = readFileSync('src/styles/variables.css', 'utf8');
    const baseStyles = readFileSync('src/styles/base.css', 'utf8');
    const headingDefaults = baseStyles.slice(0, baseStyles.indexOf('TYPOGRAPHY UTILITY CLASSES'));

    expect(variables).toContain('--text-base: 0.75rem;');
    expect(variables).toContain('--type-body-sm: var(--text-xs);');
    expect(variables).toContain('--type-body: var(--text-sm);');
    expect(headingDefaults).not.toMatch(/h2\s*\{\s*font-size:/);
  });

  it('uses the compact illustrated empty state instead of an inflated heading', () => {
    const { container } = render(
      <EmptyState title="Không có dữ liệu" description="Chưa có dữ liệu để hiển thị." size="sm" />,
    );

    expect(screen.getByText('Không có dữ liệu')).toHaveClass('text-sm');
    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      '/images/empty-states/payroll-empty-state-illustration.png',
    );
  });
});
