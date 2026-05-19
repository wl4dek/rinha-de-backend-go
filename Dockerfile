FROM golang:1.26-alpine AS build
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server
RUN go build -o ann-service ./cmd/ann-service
RUN go build -o build-index ./cmd/build-index
RUN go build -o lb ./cmd/lb
RUN ./build-index --references /app/references.json.gz --output /app/ivf_data

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root
COPY --from=build /app/server .
COPY --from=build /app/ann-service .
COPY --from=build /app/lb .
COPY --from=build /app/ivf_data ./ivf_data
COPY --from=build /app/references.json.gz .
