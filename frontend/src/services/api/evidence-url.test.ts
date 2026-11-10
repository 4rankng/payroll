import { describe, expect, it } from 'vitest';

import { assetService } from './asset.service';
import { ledgerService } from './ledger.service';

// PAYR-37: validateEvidenceUrl chỉ chấp nhận http/https — chặn javascript:,
// data:, vbscript: và các scheme khác. Hai service dùng chung contract này.
const validators = [
  { name: 'assetService', validate: (url: string) => assetService.validateEvidenceUrl(url) },
  { name: 'ledgerService', validate: (url: string) => ledgerService.validateEvidenceUrl(url) },
];

for (const { name, validate } of validators) {
  describe(`${name}.validateEvidenceUrl`, () => {
    it('accepts http and https URLs', () => {
      expect(validate('https://example.com/evidence.png').isValid).toBe(true);
      expect(validate('http://example.com/evidence.png').isValid).toBe(true);
      expect(validate('  https://example.com/x  ').isValid).toBe(true);
    });

    it('treats an empty URL as valid (optional field)', () => {
      expect(validate('').isValid).toBe(true);
      expect(validate('   ').isValid).toBe(true);
    });

    it('rejects javascript:, data: and vbscript: URLs', () => {
      expect(validate('javascript:alert(1)').isValid).toBe(false);
      expect(validate('JavaScript:alert(1)').isValid).toBe(false);
      expect(validate('data:text/html,<script>alert(1)</script>').isValid).toBe(false);
      expect(validate('vbscript:msgbox(1)').isValid).toBe(false);
    });

    it('rejects non-http protocols and malformed URLs', () => {
      expect(validate('ftp://example.com/file').isValid).toBe(false);
      expect(validate('file:///etc/passwd').isValid).toBe(false);
      expect(validate('not a url').isValid).toBe(false);
      expect(validate('example.com/x').isValid).toBe(false);
    });

    it('returns an error message when rejecting', () => {
      expect(validate('javascript:alert(1)').error).toBeTruthy();
      expect(validate('not a url').error).toBeTruthy();
    });
  });
}
