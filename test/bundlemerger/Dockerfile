FROM golang:1.22-alpine AS builder

# Set the working directory
WORKDIR /go/src/build

# Accept build arguments
ARG BUILD_TAGS

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -tags "$BUILD_TAGS" -o /go/bin/servicebinary

FROM alpine:3.21

# Install curl (healthcheck), create a user to run the service
RUN apk add --no-cache curl && \
    adduser -D -g '' appuser

USER appuser

COPY --from=builder /go/bin/servicebinary /servicebinary

# Expose the port the gRPC service listens on
EXPOSE 50051

# Expose the port for the health check
EXPOSE 80

HEALTHCHECK CMD curl --fail http://localhost:80/sequencer-testserver/health || exit 1

ENTRYPOINT ["/servicebinary"]
