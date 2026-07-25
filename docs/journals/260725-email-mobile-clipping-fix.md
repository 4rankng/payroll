# Email Mobile Clipping Fix

**Date**: 2026-07-25 10:48 +08
**Severity**: Medium
**Component**: Email branding / public banner rendering
**Status**: Resolved

## What Happened

Gmail on iOS showed the bottom of a password-reset email behind its fixed reply controls, with no remaining scroll range to reveal it. The banner was also wider than the content card and separated from it by an abnormally large gap.

The root cause was in `cloneMessageWithPublicBanner`: it was detaching an existing banner out of a table-based email layout and re-inserting it above the message body. That created extra vertical space and changed the document flow in the exact place mobile mail clients are least forgiving.

## Technical Details

- `cloneMessageWithPublicBanner` now preserves the first existing banner inside its original table cell for complete templates.
- Duplicate or malformed banner references are still normalized to the canonical public asset.
- Fragment bodies still go through the shell path, which inserts the canonical banner once.
- Regression coverage was added for banner placement so the canonical image stays inside the original table layout instead of being detached into a separate wrapper.
- Browser rendering verified the complete email at 390×844 and confirmed normal vertical scrolling with no horizontal overflow at 320×568. The original Gmail iOS screenshot was not re-tested in the live app.

## Root Cause Analysis

The branding normalizer assumed every complete message could tolerate the same banner insertion strategy. Table-based email markup depends on placement for sizing and spacing; moving the image out of its original cell left an empty row and added a second wrapper above the table.

## Lessons Learned

- Do not detach elements from existing email tables unless the layout has been tested in the target clients.
- Canonicalization has to preserve placement, not just content.
- Mobile email clients are a separate rendering environment, not a smaller browser.

## Next Steps

No API changes were needed and no deploy was performed for this journaled fix. The only follow-up is to keep the regression in place and avoid reintroducing banner detachment in future email branding changes.
