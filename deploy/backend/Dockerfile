# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git (no need for gcc/musl-dev if using CGO_ENABLED=0)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build static binary (fully static, no C dependencies)
# CGO_ENABLED=0 produces fully static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags '-w -s' \
    -o server ./cmd/server

# Runtime stage - use alpine (small, includes CA certs)
FROM alpine:3.20

# Install CA certificates and curl (for health checks)
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary
COPY --from=builder /app/server /usr/local/bin/server

# Create non-root user
RUN addgroup -g 1000 -S retailpos && \
    adduser -u 1000 -S retailpos -G retailpos && \
    chown retailpos:retailpos /usr/local/bin/server

USER retailpos

EXPOSE 8080

CMD ["/usr/local/bin/server"]
