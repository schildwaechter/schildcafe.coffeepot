# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code
COPY main.go .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o coffeepot main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS (if needed in future)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/coffeepot .

# Expose port (default 8080, can be overridden via PORT env var)
EXPOSE 8080

# Run the application
CMD ["./coffeepot"]
