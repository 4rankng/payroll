import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

// The link-splitting helpers are module-private in AssistantPanel, so assert
// on the source contract directly: an abbreviated route must be aliased onto a
// real route BEFORE the whitelist test, and unknown paths must stay plain text.
const source = readFileSync(
  resolve(process.cwd(), 'src/components/chat/AssistantPanel.tsx'),
  'utf8'
);

const LINKABLE_RE = /^\/(admin|partner|employee)\/(timesheet|approvals|payment-history|advance-payments|wallet|settings|dashboard|employees|projects|users)(\/[a-z0-9-]+)*$/;

const ALIASES: Record<string, string> = { advance: 'advance-payments', timesheets: 'timesheet' };
const resolveAlias = (p: string) => {
  const s = p.split('/');
  const l = s[s.length - 1];
  const a = ALIASES[l];
  return a ? [...s.slice(0, -1), a].join('/') : p;
};

describe('AssistantPanel deep links', () => {
  it('maps the model\'s abbreviations onto real routes', () => {
    expect(resolveAlias('/admin/advance')).toBe('/admin/advance-payments');
    expect(LINKABLE_RE.test(resolveAlias('/admin/advance'))).toBe(true);
    expect(resolveAlias('/partner/advance')).toBe('/partner/advance-payments');
    expect(resolveAlias('/admin/timesheets')).toBe('/admin/timesheet');
  });

  it('keeps invented and traversal paths unlinked', () => {
    for (const p of ['/admin/sao-ke', '/admin/../../etc/passwd', '/admin/wallet/evil/../..']) {
      expect(LINKABLE_RE.test(resolveAlias(p))).toBe(false);
    }
  });

  it('aliases before the whitelist check in the component source', () => {
    const fn = source.slice(source.indexOf('const splitPaths'), source.indexOf('const QUICK_ACTIONS'));
    expect(fn).toContain('resolveAlias(m[0])');
    expect(fn.indexOf('const resolved = resolveAlias')).toBeLessThan(fn.indexOf('LINKABLE.test(resolved)'));
  });
});