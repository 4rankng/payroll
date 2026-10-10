---
title: "Fix frontend build warnings: empty vendor chunk and 1000kB chunk warning"
date: 2026-10-10
summary: "vite manualChunks object form missed CJS react/react-dom (0-byte vendor, react-dom in ui); switched to path-based function form and aligned chunkSizeWarningLimit to the 3MB SW precache budget"
---

# Fix frontend build warnings: empty vendor chunk and 1000kB chunk warning

## What happened
- `pnpm build` (vite 6.4.3) emitted two warnings: `Generated an empty chunk: "vendor"` and `(!) Some chunks are larger than 1000 kB`.
- Evidence from the built dist: `vendor.*.js` was a 0-byte file; `Minified React error` (react-dom) lived in `ui.BEDs_yZD.js` (238 kB), not vendor; `AxiosError` sat correctly in `utils.*.js`. So the object-form `manualChunks` entry for `['react', 'react-dom']` matched nothing that survived into the graph — react/react-dom reach the bundle through CJS interop and the `react/jsx-runtime` subpath, which the bare-package object form does not capture.

## Decision
- `frontend/vite.config.ts`: replaced object-form `manualChunks` with the function form matching resolved third-party package paths (same chunk sets as before: vendor, ui, icons, utils, charts, pdfmake). Path matching also survives pnpm's `.pnpm/` layout and CJS proxy ids.
- `chunkSizeWarningLimit: 1000 -> 3072`, aligned with `injectManifest.maximumFileSizeToCacheInBytes` (3 MB for pdfMake): past that size a chunk silently drops out of SW precaching, which is the failure worth warning on. pdfmake (~2.2 MB engine+fonts, lazy-loaded) stays under it by design.

## Verification
- Rebuild: `vendor.*.js` now 143.99 kB (react+react-dom), `ui` dropped 238 -> 102 kB, zero 0-byte files in `dist/assets`, no warnings in build output, exit 0. SW precache 185 entries / ~7.8 MB.
- `pnpm lint` (eslint + type-check) and backend `go build ./... && go vet ./...` were already green before the change and re-verified clean after.

## Next steps
- The 1.5 MB entry chunk (`index.*.js`) is the known route-level code-splitting follow-up already listed in frontend/CLAUDE.md next steps; not addressed here.
- CI bundle gate stays 25 MB total (unchanged user decision).

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
