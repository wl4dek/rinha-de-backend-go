FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ann-service ./cmd/ann-service
# IF ivf_data is not included in the build context, you can build it in a separate stage and copy it to the final image
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build-index ./cmd/build-index
RUN ./build-index --references /app/references.json.gz --output /app/ivf_data

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root
COPY --from=builder /app/server .
COPY --from=builder /app/ann-service .
COPY  ./ivf_data ./ivf_data
COPY references.json.gz ./
EXPOSE 8080 8090
