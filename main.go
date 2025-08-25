package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var users = []User{
	{ID: 1, Name: "Alice Johnson", Email: "alice@example.com"},
	{ID: 2, Name: "Bob Smith", Email: "bob@example.com"},
	{ID: 3, Name: "Charlie Brown", Email: "charlie@example.com"},
}

// Metrics instruments
var (
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	activeUsers     metric.Int64UpDownCounter
	errorCounter    metric.Int64Counter
)

func main() {
	// Initialize OpenTelemetry
	shutdown := setupInstrumentation("instrumentation-test")
	defer shutdown()

	// Initialize metrics instruments
	initMetrics()

	// Set up HTTP handlers with OpenTelemetry instrumentation
	mux := http.NewServeMux()
	mux.Handle("/", otelhttp.NewHandler(http.HandlerFunc(homeHandler), "home"))
	mux.Handle("/health", otelhttp.NewHandler(http.HandlerFunc(healthHandler), "health"))
	mux.Handle("/users", otelhttp.NewHandler(http.HandlerFunc(usersHandler), "users"))
	mux.Handle("/users/", otelhttp.NewHandler(http.HandlerFunc(userHandler), "user"))
	mux.Handle("/slow", otelhttp.NewHandler(http.HandlerFunc(slowHandler), "slow"))
	mux.Handle("/error", otelhttp.NewHandler(http.HandlerFunc(errorHandler), "error"))

	appLogger.Info("Server starting", "port", 8080)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		appLogger.Error("Server failed to start", "error", err)
	}
}

// initMetrics initializes all metric instruments
func initMetrics() {
	var err error

	requestCounter, err = appMeter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		appLogger.Error("Failed to create request counter", "error", err)
	}

	requestDuration, err = appMeter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		appLogger.Error("Failed to create request duration histogram", "error", err)
	}

	activeUsers, err = appMeter.Int64UpDownCounter(
		"active_users",
		metric.WithDescription("Number of active users"),
	)
	if err != nil {
		appLogger.Error("Failed to create active users counter", "error", err)
	}

	errorCounter, err = appMeter.Int64Counter(
		"http_errors_total",
		metric.WithDescription("Total number of HTTP errors"),
	)
	if err != nil {
		appLogger.Error("Failed to create error counter", "error", err)
	}

	appLogger.Info("Metrics instruments initialized")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create a span for this operation
	ctx, span := appTracer.Start(ctx, "home_handler")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("user_agent", r.UserAgent()),
	)

	appLogger.InfoContext(ctx, "Home endpoint accessed",
		"method", r.Method,
		"url", r.URL.String(),
		"user_agent", r.UserAgent(),
	)

	response := Response{
		Message: "Welcome to the instrumentation test API",
		Data: map[string]string{
			"version": "1.0.0",
			"status":  "running",
		},
	}

	// Record metrics
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", "/"),
		attribute.String("status", "200"),
	))

	duration := time.Since(start).Seconds()
	requestDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", "/"),
	))

	writeJSONResponse(w, http.StatusOK, response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create a span for this operation
	ctx, span := appTracer.Start(ctx, "health_check")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
	)

	appLogger.InfoContext(ctx, "Health check endpoint accessed")

	response := Response{
		Message: "Service is healthy",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"uptime":    "running",
		},
	}

	// Record metrics
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", "/health"),
		attribute.String("status", "200"),
	))

	duration := time.Since(start).Seconds()
	requestDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", "/health"),
	))

	writeJSONResponse(w, http.StatusOK, response)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create a span for this operation
	ctx, span := appTracer.Start(ctx, "users_handler")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
	)

	appLogger.InfoContext(ctx, "Users endpoint accessed",
		"method", r.Method,
		"user_count", len(users),
	)

	var statusCode int
	var endpoint = "/users"

	switch r.Method {
	case http.MethodGet:
		span.SetAttributes(attribute.String("operation", "list_users"))

		response := Response{
			Message: "Users retrieved successfully",
			Data:    users,
		}
		statusCode = http.StatusOK
		writeJSONResponse(w, statusCode, response)

		appLogger.InfoContext(ctx, "Users listed successfully", "count", len(users))

	case http.MethodPost:
		span.SetAttributes(attribute.String("operation", "create_user"))

		var newUser User
		if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid request body")

			appLogger.ErrorContext(ctx, "Failed to decode user request", "error", err)

			response := Response{
				Message: "Invalid request body",
				Error:   err.Error(),
			}
			statusCode = http.StatusBadRequest
			writeJSONResponse(w, statusCode, response)

			// Record error metrics
			errorCounter.Add(ctx, 1, metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("endpoint", endpoint),
				attribute.String("error_type", "decode_error"),
			))
			return
		}

		newUser.ID = len(users) + 1
		users = append(users, newUser)

		span.SetAttributes(
			attribute.String("user.name", newUser.Name),
			attribute.String("user.email", newUser.Email),
			attribute.Int("user.id", newUser.ID),
		)

		response := Response{
			Message: "User created successfully",
			Data:    newUser,
		}
		statusCode = http.StatusCreated
		writeJSONResponse(w, statusCode, response)

		appLogger.InfoContext(ctx, "User created successfully",
			"user_id", newUser.ID,
			"user_name", newUser.Name,
			"user_email", newUser.Email,
		)

		// Update active users metric
		activeUsers.Add(ctx, 1)

	default:
		span.SetStatus(codes.Error, "Method not allowed")

		appLogger.WarnContext(ctx, "Method not allowed", "method", r.Method)

		response := Response{
			Message: "Method not allowed",
			Error:   "Only GET and POST methods are supported",
		}
		statusCode = http.StatusMethodNotAllowed
		writeJSONResponse(w, statusCode, response)

		// Record error metrics
		errorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", endpoint),
			attribute.String("error_type", "method_not_allowed"),
		))
	}

	// Record metrics
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
		attribute.String("status", fmt.Sprintf("%d", statusCode)),
	))

	duration := time.Since(start).Seconds()
	requestDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
	))
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create a span for this operation
	ctx, span := appTracer.Start(ctx, "user_handler")
	defer span.End()

	idStr := r.URL.Path[len("/users/"):]

	// Add span attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("user.id_string", idStr),
	)

	appLogger.InfoContext(ctx, "User lookup endpoint accessed",
		"method", r.Method,
		"user_id_string", idStr,
	)

	var statusCode int
	var endpoint = "/users/{id}"

	id, err := strconv.Atoi(idStr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid user ID")

		appLogger.ErrorContext(ctx, "Invalid user ID format",
			"user_id_string", idStr,
			"error", err,
		)

		response := Response{
			Message: "Invalid user ID",
			Error:   "User ID must be a number",
		}
		statusCode = http.StatusBadRequest
		writeJSONResponse(w, statusCode, response)

		// Record error metrics
		errorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", endpoint),
			attribute.String("error_type", "invalid_id"),
		))

		// Record request metrics
		requestCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", endpoint),
			attribute.String("status", fmt.Sprintf("%d", statusCode)),
		))

		duration := time.Since(start).Seconds()
		requestDuration.Record(ctx, duration, metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", endpoint),
		))
		return
	}

	span.SetAttributes(attribute.Int("user.id", id))

	for _, user := range users {
		if user.ID == id {
			span.SetAttributes(
				attribute.String("user.name", user.Name),
				attribute.String("user.email", user.Email),
				attribute.Bool("user.found", true),
			)

			appLogger.InfoContext(ctx, "User found successfully",
				"user_id", id,
				"user_name", user.Name,
				"user_email", user.Email,
			)

			response := Response{
				Message: "User found",
				Data:    user,
			}
			statusCode = http.StatusOK
			writeJSONResponse(w, statusCode, response)

			// Record request metrics
			requestCounter.Add(ctx, 1, metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("endpoint", endpoint),
				attribute.String("status", fmt.Sprintf("%d", statusCode)),
			))

			duration := time.Since(start).Seconds()
			requestDuration.Record(ctx, duration, metric.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("endpoint", endpoint),
			))
			return
		}
	}

	span.SetAttributes(attribute.Bool("user.found", false))

	appLogger.WarnContext(ctx, "User not found", "user_id", id)

	response := Response{
		Message: "User not found",
		Error:   fmt.Sprintf("No user found with ID %d", id),
	}
	statusCode = http.StatusNotFound
	writeJSONResponse(w, statusCode, response)

	// Record request metrics
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
		attribute.String("status", fmt.Sprintf("%d", statusCode)),
	))

	duration := time.Since(start).Seconds()
	requestDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
	))
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create a span for this operation
	ctx, span := appTracer.Start(ctx, "slow_operation")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
	)

	appLogger.InfoContext(ctx, "Slow operation endpoint accessed")

	// Simulate slow operation
	sleepDuration := time.Duration(rand.Intn(3)+1) * time.Second

	span.SetAttributes(attribute.Float64("operation.sleep_duration_seconds", sleepDuration.Seconds()))

	appLogger.InfoContext(ctx, "Starting slow operation",
		"sleep_duration_seconds", sleepDuration.Seconds(),
	)

	// Create a child span for the sleep operation
	_, sleepSpan := appTracer.Start(ctx, "sleep_operation")
	sleepSpan.SetAttributes(attribute.Float64("sleep.duration_seconds", sleepDuration.Seconds()))

	time.Sleep(sleepDuration)
	sleepSpan.End()

	appLogger.InfoContext(ctx, "Slow operation completed",
		"sleep_duration_seconds", sleepDuration.Seconds(),
	)

	response := Response{
		Message: "Slow operation completed",
		Data: map[string]interface{}{
			"duration_seconds": sleepDuration.Seconds(),
			"timestamp":        time.Now().Unix(),
		},
	}

	// Record metrics
	statusCode := http.StatusOK
	endpoint := "/slow"

	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
		attribute.String("status", fmt.Sprintf("%d", statusCode)),
	))

	totalDuration := time.Since(start).Seconds()
	requestDuration.Record(ctx, totalDuration, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
	))

	writeJSONResponse(w, statusCode, response)
}

func errorHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create a span for this operation
	ctx, span := appTracer.Start(ctx, "error_simulation")
	defer span.End()

	// Add span attributes
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
	)

	appLogger.InfoContext(ctx, "Error simulation endpoint accessed")

	var statusCode int
	var endpoint = "/error"

	// Simulate random errors
	willError := rand.Intn(2) == 0
	span.SetAttributes(attribute.Bool("simulation.will_error", willError))

	if willError {
		// Simulate an error condition
		errorMsg := "Something went wrong in the system"

		span.RecordError(fmt.Errorf(errorMsg))
		span.SetStatus(codes.Error, errorMsg)
		span.SetAttributes(attribute.String("error.type", "simulated_error"))

		appLogger.ErrorContext(ctx, "Simulated internal server error",
			"error_message", errorMsg,
			"simulation", true,
		)

		response := Response{
			Message: "Internal server error",
			Error:   errorMsg,
		}
		statusCode = http.StatusInternalServerError
		writeJSONResponse(w, statusCode, response)

		// Record error metrics
		errorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("endpoint", endpoint),
			attribute.String("error_type", "simulated_error"),
		))
	} else {
		span.SetAttributes(attribute.String("result", "success"))

		appLogger.InfoContext(ctx, "Error endpoint accessed successfully (no error simulated)")

		response := Response{
			Message: "Error endpoint accessed successfully",
			Data: map[string]string{
				"note": "This endpoint randomly returns errors",
			},
		}
		statusCode = http.StatusOK
		writeJSONResponse(w, statusCode, response)
	}

	// Record request metrics
	requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
		attribute.String("status", fmt.Sprintf("%d", statusCode)),
	))

	duration := time.Since(start).Seconds()
	requestDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("endpoint", endpoint),
	))
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log encoding errors but don't change the response since headers are already sent
		if appLogger != nil {
			appLogger.Error("Failed to encode JSON response", "error", err, "status_code", statusCode)
		}
	}
}
