# Production packaging

`npm run build` produces the static SPA in `build/`. Serve that directory and the Go API from one origin:

- static UI and `index.html` fallback: `/`
- application API: `/api/v1/*`

The browser client intentionally uses only relative `/api/v1` URLs. The Vite proxy to `127.0.0.1:8080` is development-only; production must mount or reverse-proxy the Go handler at the same-origin `/api/v1` prefix. Do not expose the loopback backend directly.

Preview and PDF consume `src/lib/renderer/v1/render.ts`. Renderer versions are immutable: breaking scene changes require a new directory/version so an export worker can pin the same renderer used for Preview.

The renderer exposes `window.__PRESENTATOR_RENDER_READY__`. Chromium must await this promise before screenshot/PDF capture. It resolves only after fonts and every image load with `naturalWidth > 0`; a missing/corrupt asset rejects with `asset_load_failed:<assetId>`. Export workers must treat rejection as terminal failure and must not publish a partial artifact. Asset IDs resolve through the allowlisted same-origin `/api/v1/assets/{id}` contract unless an explicit trusted asset map is supplied.
