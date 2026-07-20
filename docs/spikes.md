# Architecture spike results

## PocketBase persistence/job gate

Status: **conditional pass for persistence, fail for durable job claiming until proven against a pinned PocketBase version**.

The runnable slice keeps `Store`, `PredictorXPort`, and `PDFPort` behind boundaries. PocketBase can persist projects, immutable deck revisions, assets, and job records. A production worker must not use read-then-update claiming: it needs an atomic compare-and-set/transaction, lease owner, lease expiry, heartbeat, and crash reconciliation. No PocketBase binary/version or production topology is committed yet, so this repository deliberately ships an in-memory reference store and records the durable-queue gate instead of claiming an unverified guarantee. If the selected PocketBase SDK cannot provide atomic claim, use PostgreSQL/queue for jobs while retaining PocketBase files/auth.

## PredictorX transport gate

Status: **product contract passes; live transport blocked by missing endpoint/auth/workflow profile**.

`PredictorXPort` proves the application boundary and the fixture adapter exercises queued → running → succeeded → candidate/apply. Live integration requires a server-side endpoint, auth mechanism, launch/status/result/cancel routes and timeout limits. Secrets are intentionally absent. Contract failures map to stable product errors; raw provider output must be size-limited and validated before candidate persistence.

## Preview/PDF renderer gate

Status: **worker/immutable-revision flow passes; fidelity gate remains open for shared Chromium renderer**.

`PDFPort` and `MinimalPDF` prove asynchronous export and emit a syntactically structured multi-page PDF without dependencies. It is not the production fidelity renderer. Production must render the same versioned HTML/scene package used by Preview in pinned Chromium, with bundled fonts/assets and golden pixel/PDF tests for text, image crop, rectangle, ellipse and line. This cannot be completed before the frontend render package exists.
