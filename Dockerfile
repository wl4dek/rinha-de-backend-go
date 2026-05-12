FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN ./build-index --references /app/references.json.gz --output /app/ivf_data

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root
COPY --from=builder /app/server .
COPY --from=builder /app/ann-service .
COPY --from=builder /app/ivf_data ./ivf_data
COPY references.json.gz ./
EXPOSE 8080 8090
