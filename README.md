# Presentator

Backend vertical slice for an AI-assisted presentation editor.

```sh
go run ./cmd/presentator
# GET http://127.0.0.1:8080/api/v1/healthz
```

Contracts live in `api/openapi.yaml`, `api/deck.schema.json`, and `api/fixtures/`.
The current adapters are deterministic local implementations; see `docs/spikes.md` before configuring PocketBase, PredictorX, or Chromium in production.

Verification:

```sh
go test ./...
go test -race ./...
go vet ./...
```
