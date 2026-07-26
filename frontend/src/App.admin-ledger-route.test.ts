import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Admin ledger responsive route', () => {
  it('keeps the desktop transactions page while using the ledger page on mobile', () => {
    const appSource = readFileSync(resolve(process.cwd(), 'src/App.tsx'), 'utf8');

    expect(appSource).toContain(
      '<Route path="ledger" element={<ResponsivePage desktopComponent={TransactionsPage} mobileComponent={LedgerEntriesPageMobile} />} />',
    );
  });
});
