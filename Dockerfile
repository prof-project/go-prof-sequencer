FROM golang:1.23-alpine AS builder

# Set the working directory
WORKDIR /go/src/build

# Accept build arguments
ARG BUILD_TAGS

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -tags "$BUILD_TAGS" -o /go/bin/servicebinary

# Install upx and compress the compiled binary
RUN apk add --no-cache upx && upx -q -9 /go/bin/servicebinary

FROM alpine:3.21

# Install curl (healthcheck), create a user to run the service
RUN apk add --no-cache curl && \
    adduser -D -g '' appuser && \
    mkdir -p /app/logs && \
    chown -R appuser:appuser /app/logs

USER appuser

COPY --from=builder /go/bin/servicebinary /servicebinary

# Expose the port the http service listens on
EXPOSE 80

# Expose the port for the Prometheus metrics
EXPOSE 8080

HEALTHCHECK CMD curl --fail http://localhost:80/sequencer/health || exit 1

ENTRYPOINT ["/servicebinary"]
