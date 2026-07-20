# Production packaging

`npm run build` produces the static SPA in `build/`. Serve that directory and the Go API from one origin:

- static UI and `index.html` fallback: `/`
- application API: `/api/v1/*`

The browser client intentionally uses only relative `/api/v1` URLs. The Vite proxy to `127.0.0.1:8080` is development-only; production must mount or reverse-proxy the Go handler at the same-origin `/api/v1` prefix. Do not expose the loopback backend directly.

Preview and PDF consume `src/lib/renderer/v1/render.ts`. Renderer versions are immutable: breaking scene changes require a new directory/version so an export worker can pin the same renderer used for Preview.

Run `npm run build:renderer` during packaging. The Go worker executes
`scripts/render-pdf.mjs`, which imports that exact immutable renderer build and
uses exact `playwright@1.60.0` / its Chromium revision. Provision with
`npx playwright install --with-deps chromium`; startup readiness verifies the
renderer version, Playwright pin and browser version before accepting traffic.

The renderer exposes `window.__PRESENTATOR_RENDER_READY__`. Chromium awaits this
promise before capture. It resolves only after the requested fonts and every
image load with `naturalWidth > 0`; missing/corrupt assets reject with
`asset_load_failed:<assetId>` and no partial artifact is published. The browser
uses the allowlisted same-origin project asset API; the export worker supplies
an ownership-checked data-URI map. Arbitrary external asset URLs are forbidden.
