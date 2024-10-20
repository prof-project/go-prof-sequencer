FROM golang:1.22-alpine AS builder

# Set the working directory
WORKDIR /go/src/build

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
     GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -o /go/bin/servicebinary

FROM alpine
COPY --from=builder /go/bin/servicebinary /servicebinary
ENTRYPOINT ["/servicebinary"]
