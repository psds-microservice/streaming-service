# recording-service and session-manager-service from repo. helpy/infra via GOMODCACHE.
FROM golang:1.26-alpine AS builder
ENV GOMODCACHE=/go/pkg/mod
WORKDIR /app
COPY helpy /go/pkg/mod/github.com/psds-microservice/helpy@v0.1.0
COPY infra /go/pkg/mod/github.com/psds-microservice/infra@v0.0.3
COPY streaming-service ./streaming-service
WORKDIR /app/streaming-service
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /streaming-service ./cmd/streaming-service

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /streaming-service .
COPY --from=builder /app/streaming-service/database ./database
EXPOSE 8090
CMD ["./streaming-service"]
