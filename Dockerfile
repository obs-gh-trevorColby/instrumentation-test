# Build stage
FROM golang:1.23-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code including OpenTelemetry setup
COPY main.go otel.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o instrumentation-test .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/instrumentation-test .

# Expose port 8080
EXPOSE 8080

# Set environment variables for OpenTelemetry
ENV OTEL_EXPORTER_OTLP_ENDPOINT=""
ENV OTEL_EXPORTER_OTLP_BEARER_TOKEN=""

# Run the application
CMD ["./instrumentation-test"]
