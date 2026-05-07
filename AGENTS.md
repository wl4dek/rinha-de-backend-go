# AGENTS.md

## Project

Go 1.26.0, module `rinha-de-backend`. Single binary initially, now with ANN service.

## Structure

- `cmd/server/main.go` — API entrypoint, starts HTTP server on port 8080
- `cmd/ann-service/main.go` — ANN service entrypoint, HNSW index on port 8090
- `internal/frauds_core/` — fraud scoring logic (14-dim vectors, rule-based + ANN)
- `internal/ann/` — HNSW ANN service (coder/hnsw library)

## Build & Run

```
go build ./cmd/server
go build ./cmd/ann-service
```

API: `go run ./cmd/server`
ANN: `go run ./cmd/ann-service`

## Env Vars

- `PORT` (default: 8080) — API server port
- `ANN_PORT` (default: 8090) — ANN service port
- `ANN_SERVICE_URL` (default: http://localhost:8090) — URL for API to reach ANN service
- `REFERENCES_PATH` (default: ./references.json.gz) — path to reference vectors
- `INDEX_BIN_PATH` (default: ./index.bin) — HNSW index persistence file

## API Contract

- `GET /ready` — health check, returns 200 OK
- `POST /fraud-score` — receive transaction, return fraud decision with ANN-enhanced scoring

## ANN Service

- Uses `github.com/coder/hnsw` with 14-dimensional vectors
- Pre-loads from `references.json.gz` (fields: `vector`, `label`)
- Saves index to `index.bin` after first build (one-time save)
- `POST /search` — KNN search with k=5 default
- `GET /ready` — health check

## Docker

`docker-compose.yml` defines 4 services within 1 CPU, 350MB total:
- `ann-service` (0.3 CPU, 120MB)
- `api01` + `api02` (0.25 CPU, 80MB each)
- `nginx` (0.2 CPU, 70MB) — load balancer on port 9999

## Notes

- No tests yet beyond basic endpoint checks
- No lint/formatter config beyond `go fmt`
- `references.json.gz` contains 3M reference vectors
- ANN index build on first run takes time; subsequent starts load `index.bin` instantly
