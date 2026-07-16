# Admin email composer

Status: completed

## Scope

- Let an administrator choose from configured, verified outbound email senders.
- Add an admin email composer that accepts rich pasted content and raw HTML, shows a sandboxed preview with the TingTing banner, and sends recipients, CC/BCC, subject, and optional reply-to.
- Preserve the provider-side TingTing branding as the authoritative delivery behavior.

## Implementation and verification

1. Add a configured sender allow-list and sender-discovery API; reject arbitrary `From` values.
2. Make the generic email endpoint support both JSON and multipart payloads.
3. Add the composer, API hooks, route, and navigation entry.
4. Add unit coverage for sender selection and payload handling; run backend tests plus frontend type-check, lint, and production build.

## Acceptance criteria

- An admin can choose a verified configured sender instead of the `noreply@tingting.vip` default.
- Every outgoing generic email contains exactly one TingTing banner.
- Rich content copied from an external editor or raw pasted HTML is sent as HTML and can be previewed safely in the browser.
- Sending remains admin-only; unconfigured senders are rejected server-side.
