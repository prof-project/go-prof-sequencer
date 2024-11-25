FROM golang:1.22-alpine AS builder

# Set the working directory
WORKDIR /go/src/build

# Accept build arguments
ARG BUILD_TAGS

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
     GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -tags "$BUILD_TAGS" -o /go/bin/servicebinary

FROM alpine

# Install curl
RUN apk add --no-cache curl

COPY --from=builder /go/bin/servicebinary /servicebinary

# Expose the port your service listens on
EXPOSE 80

ENTRYPOINT ["/servicebinary"]
