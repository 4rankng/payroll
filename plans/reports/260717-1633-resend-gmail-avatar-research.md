# Resend Gmail Avatar Research

---
date: 2026-07-17 16:33 +08:00
status: complete
scope: TingTing logo for marketing@tingting.vip emails sent through Resend
---

## Summary

For the avatar shown inside an opened Gmail message, use a Google Account whose exact address is `marketing@tingting.vip` and set the TingTing logo as that account's profile picture. Resend documents this as its Gmail-specific avatar method even when Resend performs delivery.

For a standards-based logo across supported mailbox providers, implement BIMI. Gmail requires a Common Mark Certificate (CMC) or Verified Mark Certificate (VMC); an unauthenticated standalone BIMI SVG is insufficient for Gmail.

## Findings

### Immediate Gmail route

1. Create or sign in to a Google Account using `marketing@tingting.vip`.
2. Upload a square TingTing logo as the Google Account profile picture.
3. Send through Resend with the exact From address `marketing@tingting.vip`.
4. Test with a separate Gmail recipient and allow time for Google caches to refresh.

Resend notes this avatar appears in the Gmail mobile app and inside opened messages on desktop. It is provider-specific and is not guaranteed in every inbox or inbox-list view.

### BIMI route

Requirements:

- Resend domain verified with aligned SPF/DKIM.
- DMARC `p=quarantine` or `p=reject` and `pct=100`.
- Square SVG Tiny P/S logo hosted over HTTPS.
- CMC for Gmail logo display when eligible through established logo use, or VMC for a trademarked logo and Gmail's verification checkmark.
- `default._bimi.tingting.vip` TXT record pointing to the logo and certificate.

### Live DNS state

Checked 2026-07-17:

- Application constant already uses `marketing@tingting.vip` and the sender name `Ting Ting Software Solution`; no application code change is needed for the Google profile association.
- Root SPF authorizes Resend: `v=spf1 include:_spf.resend.com ~all`.
- Resend return path exists at `send.tingting.vip` with Amazon SES SPF/MX.
- DMARC exists but uses `p=none`, which does not satisfy BIMI.
- No `default._bimi.tingting.vip` TXT record is currently published.
- Inbound MX uses ImprovMX; this does not itself block Resend or BIMI.

Do not strengthen DMARC until a received test message shows `dmarc=pass` and all other legitimate senders for `tingting.vip` have been inventoried.

## Recommendation

Use the Google Account profile-picture route now because it directly targets the avatar shown in the supplied Gmail screenshot and does not require a certificate. Add BIMI later if consistent brand identity across supported inboxes justifies certificate cost and verification work.

## References

- [Resend: How do I send with an avatar?](https://resend.com/docs/knowledge-base/how-do-i-send-with-an-avatar)
- [Resend: Implementing BIMI](https://resend.com/docs/dashboard/domains/bimi)
- [Resend: Implementing DMARC](https://resend.com/docs/dashboard/domains/dmarc)
- [Google Workspace: Set up BIMI](https://support.google.com/a/answer/10911320)

## Unresolved Questions

- Whether `marketing@tingting.vip` already has a Google Account association.
- Whether the TingTing logo is trademarked or has at least one year of documented public use.
