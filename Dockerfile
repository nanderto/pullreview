# Build stage
FROM golang:1.24 AS builder

# Set working directory
WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with static linking (CGO disabled for fully static binary)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o pullreview ./cmd/pullreview

# Run stage - using distroless
FROM gcr.io/distroless/static-debian12:nonroot

# Set working directory
WORKDIR /app

# Copy binary and prompt file from builder
COPY --from=builder /build/pullreview .
COPY --from=builder /build/prompt.md .

# Set the entrypoint
ENTRYPOINT ["/app/pullreview"]
