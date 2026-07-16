import DOMPurify from 'dompurify';

export const TINGTING_EMAIL_BANNER_URL = 'https://tingting.vip/email-banner.jpg?v=20260709';

export function parseEmailRecipients(value: string): string[] {
  return value
    .split(/[;,\n]/)
    .map((address) => address.trim())
    .filter(Boolean);
}

export function plainTextFromHtml(html: string): string {
  const document = new DOMParser().parseFromString(html, 'text/html');
  return document.body.textContent?.trim() ?? '';
}

export function buildEmailPreviewDocument(html: string): string {
  const cleaned = DOMPurify.sanitize(html, {
    ADD_ATTR: ['align', 'bgcolor', 'border', 'cellpadding', 'cellspacing', 'height', 'style', 'target', 'width'],
  });
  const document = new DOMParser().parseFromString(cleaned, 'text/html');
  document.querySelectorAll('img[src*="tingting.vip/email-banner.jpg"]').forEach((image) => image.remove());
  const normalizedContent = document.body.innerHTML;
  const banner = `<div style="max-width:640px;margin:0 auto 16px;"><img src="${TINGTING_EMAIL_BANNER_URL}" alt="Ting Ting Soft" width="640" style="display:block;width:100%;max-width:640px;height:auto;border:0;border-radius:16px;"></div>`;

  return `<!doctype html><html lang="vi"><head><meta charset="utf-8"></head><body style="margin:0;padding:24px 12px;background:#f5f7fb;color:#172033;font-family:Arial,sans-serif;"><main style="max-width:640px;margin:0 auto;">${banner}<section style="background:#fff;border:1px solid #e6eaf2;border-radius:16px;padding:28px 32px;">${normalizedContent}</section></main></body></html>`;
}
