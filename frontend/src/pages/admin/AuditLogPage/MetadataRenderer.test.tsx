import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { MetadataRenderer } from './MetadataRenderer';

describe('MetadataRenderer', () => {
  it('shows backend before and after values for changed fields', () => {
    const metadata = JSON.stringify({
      changed_fields: {
        date_of_birth: {
          before: '1990-01-02',
          after: '1991-03-04',
        },
        bank_account_number: {
          before: '0123456789',
          after: '9876543210',
        },
      },
    });

    render(<MetadataRenderer metadata={metadata} />);

    expect(screen.getByText('1990-01-02')).toBeInTheDocument();
    expect(screen.getByText('1991-03-04')).toBeInTheDocument();
    expect(screen.getByText('0123456789')).toBeInTheDocument();
    expect(screen.getByText('9876543210')).toBeInTheDocument();
  });

  it('keeps rendering legacy old and new change values', () => {
    const metadata = JSON.stringify({
      changed_fields: {
        fullname: {
          old: 'Nguyễn Văn A',
          new: 'Nguyễn Văn B',
        },
      },
    });

    render(<MetadataRenderer metadata={metadata} />);

    expect(screen.getByText('Nguyễn Văn A')).toBeInTheDocument();
    expect(screen.getByText('Nguyễn Văn B')).toBeInTheDocument();
  });
});
