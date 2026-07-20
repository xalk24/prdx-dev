# Presentator

AI-assisted presentation editor with a Svelte frontend and Go API/workers.

```sh
npm ci
npm run dev

go run ./cmd/presentator
# GET http://127.0.0.1:8080/api/v1/healthz
```

Contracts live in `api/openapi.yaml`, `api/deck.schema.json`, and `api/fixtures/`.
Preview and PDF use the immutable renderer in `src/lib/renderer/v1/`.
See `docs/production.md` and `docs/spikes.md` before production configuration.

Verification:

```sh
npm run check
npm run lint
npm test -- --run
npm run build
go test ./...
go test -race ./...
go vet ./...
```
