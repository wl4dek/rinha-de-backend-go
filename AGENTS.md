# AGENTS.md

## Project

Go 1.26.0, module `rinha-de-backend`. Single binary initially, now with ANN service.

## Structure

- `cmd/server/main.go` — API entrypoint, starts HTTP server on port 8080
- `cmd/ann-service/main.go` — ANN service entrypoint, IVF index on port 8090
- `cmd/build-index/main.go` — offline tool to build IVF index from references
- `cmd/lb/main.go` — load balancer (round-robin) em Go, substitui nginx
- `internal/frauds_core/` — fraud scoring logic (14-dim vectors, rule-based + ANN)
- `internal/ann/` — IVF index (clustering + HNSW on centroids + scan inside cluster)
  - `ivf.go` — IVFIndex struct, LoadIVF, Search (max-heap top-k, 3-cluster probe)
  - `build.go` — BuildIVF, kmeansPP (k-means++ on normalized vectors)
  - `math.go` — normalize, dotProduct (loop unrolled), nearestCentroid, addVec
  - `heap.go` — maxHeap (container/heap) for top-k selection
  - `mmap.go` — mmapFloat32 (MADV_SEQUENTIAL), readUint32File, readFloat32Vectors
  - `handler.go` — HTTP handlers (/search, /ready)
  - `types.go` — request/response types, const Dimensions

## Build & Run

```
go build ./cmd/server
go build ./cmd/ann-service
go build ./cmd/build-index
go build ./cmd/lb
```

Build IVF index first:
```
go run ./cmd/build-index --references ./references.json.gz --output ./ivf_data
```

API: `go run ./cmd/server`
ANN: `go run ./cmd/ann-service`

## Env Vars

- `PORT` (default: 8080) — API server port
- `ANN_PORT` (default: 8090) — ANN service port
- `ANN_SERVICE_URL` (default: http://localhost:8090) — URL for API to reach ANN service
- `IVF_DATA_PATH` (default: ./ivf_data) — path to precomputed IVF index files

## API Contract

- `GET /ready` — health check, returns 200 OK
- `POST /fraud-score` — receive transaction, return fraud decision with ANN-enhanced scoring

## ANN Service

- IVF index with 500 clusters, using `github.com/coder/hnsw` for centroid indexing
- Pre-built offline by `cmd/build-index` (k-means++ on 50K sample, then assigns all 3M)
  - All vectors pre-normalized to unit length at build time
- At runtime: loads centroids in HNSW + cluster assignments + labels (~23 MB RSS, ~11 MB heap)
- Vectors stored cluster-ordered in vectors.bin (same cluster = contiguous on disk)
- Vectors accessed via mmap + MADV_SEQUENTIAL (168 MB virtual, sequential cluster scan loads ~0.3 MB per search)
- Search probes 3 nearest clusters, uses max-heap (O(n log k)) for top-k, dot product on normalized vectors
- `POST /search` — KNN search with k=5 default
- `GET /ready` — health check

## Docker

`docker-compose.yml` defines 4 services within 1 CPU, 350MB total:
- `ann-service` (0.3 CPU, 120MB)
- `api01` + `api02` (0.25 CPU, 70MB each)
- `lb` (0.15 CPU, 20MB) — load balancer (round-robin) em Go, porta 9999

## Notes

- No tests yet beyond basic endpoint checks
- No lint/formatter config beyond `go fmt`
- `references.json.gz` contains 3M reference vectors (14-dim float32, ~48 MB gzipped)
- IVF index is built at Docker build time via `cmd/build-index`
- Binary files in `ivf_data/`: ivf.bin (header), vectors.bin (168 MB), centroids.bin (28 KB), labels.bin (3 MB), cluster_bounds.bin (2 KB)
- build-index needs ~500 MB RAM temporarily (loads 3M refs + k-means), ann-service only ~23 MB
- `github.com/TFMV/quiver` replaced with direct `github.com/coder/hnsw`
- All vectors pre-normalized at build time so search uses simple dot product (no cosine similarity)
- `github.com/chewxy/math32` removed — no longer needed
