import { describe, expect, it } from 'vitest';
import { buildEmailPreviewDocument, parseEmailRecipients, TINGTING_EMAIL_BANNER_URL } from './email-composer-utils';

describe('email composer utilities', () => {
  it('splits recipients pasted with common separators', () => {
    expect(parseEmailRecipients('a@example.com; b@example.com\nc@example.com')).toEqual([
      'a@example.com',
      'b@example.com',
      'c@example.com',
    ]);
  });

  it('shows one TingTing banner and removes executable preview content', () => {
    const preview = buildEmailPreviewDocument('<p>Xin chào</p><script>window.alert(1)</script>');

    expect(preview.split(TINGTING_EMAIL_BANNER_URL)).toHaveLength(2);
    expect(preview).not.toContain('<script>');
  });

  it('normalizes pasted banner images but keeps non-image references', () => {
    const preview = buildEmailPreviewDocument(`<img src="${TINGTING_EMAIL_BANNER_URL}" alt="TingTing Soft"><img src="${TINGTING_EMAIL_BANNER_URL}"><a href="https://tingting.vip/email-banner.jpg">Tải banner</a>`);

    expect(preview.split(TINGTING_EMAIL_BANNER_URL)).toHaveLength(2);
    expect(preview).toContain('href="https://tingting.vip/email-banner.jpg"');
  });
});
