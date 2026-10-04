import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Tabs, TabsList, TabsTrigger } from './tabs';

function renderTabs() {
  return render(
    <Tabs defaultValue="a">
      <TabsList>
        <TabsTrigger value="a">Cơ bản</TabsTrigger>
        <TabsTrigger value="b">Nâng cao</TabsTrigger>
      </TabsList>
    </Tabs>,
  );
}

describe('Tabs contract', () => {
  it('forwards the ref to the Radix list', () => {
    const listRef = createRef<HTMLDivElement>();
    render(
      <Tabs defaultValue="a">
        <TabsList ref={listRef}>
          <TabsTrigger value="a">Cơ bản</TabsTrigger>
        </TabsList>
      </Tabs>,
    );

    expect(listRef.current).toBeInstanceOf(HTMLDivElement);
    expect(screen.getByRole('tablist')).toBe(listRef.current);
  });

  it('active tab: card surface + text-fg-primary + shadow-xs', () => {
    renderTabs();
    const active = screen.getByRole('tab', { name: 'Cơ bản' });
    expect(active).toHaveClass(
      'data-[state=active]:bg-card',
      'data-[state=active]:text-fg-primary',
      'data-[state=active]:shadow-xs',
    );
  });

  it('inactive tab: text-fg-tertiary with utility-gray-50 hover', () => {
    renderTabs();
    const inactive = screen.getByRole('tab', { name: 'Nâng cao' });
    expect(inactive).toHaveClass('text-fg-tertiary', 'hover:bg-utility-gray-50');
  });

  it('list surface is utility-gray-100', () => {
    renderTabs();
    expect(screen.getByRole('tablist')).toHaveClass('bg-utility-gray-100', 'rounded-lg');
  });
});
