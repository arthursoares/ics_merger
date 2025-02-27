FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy go.mod and go.sum
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o ical_merger ./cmd

# Create a minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/ical_merger /app/

# Create directories for configuration and output
RUN mkdir -p /app/output

# Set the entrypoint
ENTRYPOINT ["/app/ical_merger"]