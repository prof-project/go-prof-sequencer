FROM golang:1.22-alpine AS builder

# Set the working directory
WORKDIR /go/src/build

# Accept build arguments
ARG BUILD_TAGS

RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -tags "$BUILD_TAGS" -o /go/bin/servicebinary

# Install upx
RUN apk add --no-cache upx

# Compress the compiled binary
RUN upx -q -9 /go/bin/servicebinary

FROM alpine:latest

# Install curl
RUN apk add --no-cache curl

COPY --from=builder /go/bin/servicebinary /servicebinary

# Expose the port your service listens on
EXPOSE 80

# Read secrets from files
ENV SEQUENCER_DEFAULT_ADMIN_PASSWORD_FILE=/run/secrets/sequencer_default_admin_password
ENV SEQUENCER_DEFAULT_USER1_PASSWORD_FILE=/run/secrets/sequencer_default_user1_password
ENV SEQUENCER_JWT_KEY_FILE=/run/secrets/sequencer_jwt_key

ENTRYPOINT ["/servicebinary"]
