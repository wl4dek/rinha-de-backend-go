# AGENTS.md

## Project

Go 1.26.0, module `rinha-de-backend`. Single binary, no monorepo.

## Structure

- `cmd/server/main.go` — entrypoint, starts HTTP server
- `internal/fraud_score/` — business logic (currently a stub)

## Run

```
go run ./cmd/server
```

Port via `PORT` env var (default: 8080).

## API contract (rinha-de-backend)

- `GET /ready` — health check, must return 2xx
- `POST /fraud-score` — receive transaction data, return fraud decision

## Notes

- No tests, Makefile, Dockerfile, or CI config exist yet.
- No lint or formatter config beyond `go fmt`.
- `internal/fraud_score/handle.go` is a stub — implement `HandleFraudScore`.
