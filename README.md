# OpenTelemetry Instrumentation Test

A comprehensive Go web service demonstrating OpenTelemetry instrumentation with distributed tracing, structured logging, and metrics collection.

## Features

### 🔍 **Comprehensive Observability**
- **Distributed Tracing**: Full request tracing with OpenTelemetry spans
- **Structured Logging**: JSON logs with trace correlation using slog
- **Metrics Collection**: Request counters, duration histograms, and business metrics
- **Error Tracking**: Proper error recording and monitoring

### 🚀 **API Endpoints**
- `GET /` - Welcome message with service information
- `GET /health` - Health check endpoint
- `GET /users` - List all users
- `POST /users` - Create a new user
- `GET /users/{id}` - Get user by ID
- `GET /slow` - Simulates slow operations (1-3 seconds)
- `GET /error` - Randomly returns errors for testing error tracking

### 📊 **Instrumentation Details**

#### Tracing
- Automatic HTTP request tracing via `otelhttp` middleware
- Manual spans for business operations
- Proper span attributes following OpenTelemetry semantic conventions
- Error recording with status codes

#### Logging
- Structured JSON logging with `log/slog`
- Automatic trace correlation via `otelslog` bridge
- Context-aware logging with request details

#### Metrics
- `http_requests_total` - Counter of HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Histogram of request durations
- `http_errors_total` - Counter of HTTP errors by type
- `active_users` - Up/down counter tracking user creation

## 🛠️ **Setup and Usage**

### Prerequisites
- Go 1.23 or later
- OpenTelemetry Collector (optional, for data export)

### Environment Variables
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OTLP endpoint (default: localhost:4317)
- `OTEL_EXPORTER_OTLP_BEARER_TOKEN` - Bearer token for authentication

### Running Locally

1. **Clone and build:**
   ```bash
   git clone <repository-url>
   cd instrumentation-test
   go mod download
   go build -o instrumentation-test .
   ```

2. **Run the application:**
   ```bash
   ./instrumentation-test
   ```

3. **Test the endpoints:**
   ```bash
   # Health check
   curl http://localhost:8080/health

   # List users
   curl http://localhost:8080/users

   # Create a user
   curl -X POST http://localhost:8080/users \
     -H "Content-Type: application/json" \
     -d '{"name":"John Doe","email":"john@example.com"}'

   # Get user by ID
   curl http://localhost:8080/users/1

   # Test slow endpoint
   curl http://localhost:8080/slow

   # Test error endpoint
   curl http://localhost:8080/error
   ```

### Running with Docker

1. **Build the Docker image:**
   ```bash
   docker build -t instrumentation-test .
   ```

2. **Run the container:**
   ```bash
   docker run -p 8080:8080 \
     -e OTEL_EXPORTER_OTLP_ENDPOINT="your-collector:4317" \
     -e OTEL_EXPORTER_OTLP_BEARER_TOKEN="your-token" \
     instrumentation-test
   ```

### Running Tests

```bash
go test -v
```

## 📈 **Observability Data**

The application exports telemetry data to an OpenTelemetry Collector via OTLP gRPC. When properly configured, you'll see:

### Traces
- HTTP request spans with timing and attributes
- Business operation spans (user creation, lookups)
- Error spans with proper status codes

### Logs
- Structured JSON logs with trace correlation
- Request/response logging
- Error logging with context

### Metrics
- Request rate and latency metrics
- Error rate tracking
- Business metrics (user counts)

## 🏗️ **Architecture**

### Files
- `main.go` - Main application with instrumented HTTP handlers
- `otel.go` - OpenTelemetry setup and configuration
- `main_test.go` - Comprehensive test suite
- `Dockerfile` - Container build configuration

### Dependencies
- OpenTelemetry Go SDK with OTLP exporters
- HTTP instrumentation via `otelhttp`
- Structured logging bridge via `otelslog`

## 🔧 **Development**

### Code Quality
- All code passes `go vet` and `go fmt`
- Comprehensive test coverage
- Follows Go best practices and OpenTelemetry conventions

### Adding New Endpoints
1. Create handler function with context-aware instrumentation
2. Add tracing spans with appropriate attributes
3. Include structured logging with trace correlation
4. Record relevant metrics
5. Add tests for the new functionality

This application demonstrates production-ready OpenTelemetry instrumentation patterns while maintaining code simplicity and performance.
