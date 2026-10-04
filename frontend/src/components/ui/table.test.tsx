import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from './table';

describe('table (UU restyle contract)', () => {
  it('forwards refs for every table part', () => {
    const tableRef = createRef<HTMLTableElement>();
    const headerRef = createRef<HTMLTableSectionElement>();
    const bodyRef = createRef<HTMLTableSectionElement>();
    const footerRef = createRef<HTMLTableSectionElement>();
    const rowRef = createRef<HTMLTableRowElement>();
    const headRef = createRef<HTMLTableCellElement>();
    const cellRef = createRef<HTMLTableCellElement>();
    const captionRef = createRef<HTMLTableCaptionElement>();

    render(
      <Table ref={tableRef}>
        <TableCaption ref={captionRef}>Danh sách nhân viên</TableCaption>
        <TableHeader ref={headerRef}>
          <TableRow ref={rowRef}>
            <TableHead ref={headRef}>Tên</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody ref={bodyRef}>
          <TableRow>
            <TableCell ref={cellRef}>Nguyễn Văn A</TableCell>
          </TableRow>
        </TableBody>
        <TableFooter ref={footerRef}>
          <TableRow>
            <TableCell>Tổng</TableCell>
          </TableRow>
        </TableFooter>
      </Table>,
    );

    expect(tableRef.current).toBeInstanceOf(HTMLTableElement);
    expect(headerRef.current).toBeInstanceOf(HTMLTableSectionElement);
    expect(bodyRef.current).toBeInstanceOf(HTMLTableSectionElement);
    expect(footerRef.current).toBeInstanceOf(HTMLTableSectionElement);
    expect(rowRef.current).toBeInstanceOf(HTMLTableRowElement);
    expect(headRef.current).toBeInstanceOf(HTMLTableCellElement);
    expect(cellRef.current).toBeInstanceOf(HTMLTableCellElement);
    expect(captionRef.current).toBeInstanceOf(HTMLTableCaptionElement);
  });

  it('renders the UU bridge tokens on header, rows and footer', () => {
    render(
      <Table className="min-w-0">
        <TableHeader>
          <TableRow>
            <TableHead>Tên</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow data-testid="body-row">
            <TableCell>Nhân viên</TableCell>
          </TableRow>
        </TableBody>
        <TableFooter>
          <TableRow>
            <TableCell>Tổng</TableCell>
          </TableRow>
        </TableFooter>
      </Table>,
    );

    // Root contract + caller className merge.
    expect(screen.getByRole('table')).toHaveClass('w-full', 'min-w-0');
    expect(screen.getByRole('columnheader')).toHaveClass(
      'text-fg-tertiary',
      'font-semibold',
      'text-xs',
    );
    expect(screen.getByTestId('body-row')).toHaveClass(
      'border-b',
      'border-utility-gray-200',
      'hover:bg-utility-gray-50',
      'data-[state=selected]:bg-utility-gray-100',
    );
    const footer = document.querySelector('tfoot');
    expect(footer).toHaveClass(
      'bg-utility-gray-50',
      'border-utility-gray-200',
      'typography-label-medium',
    );
  });
});
