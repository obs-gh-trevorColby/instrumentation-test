#!/usr/bin/env python3
"""
Simple Flask web application for OpenTelemetry instrumentation testing.
This application demonstrates various service boundaries that can be instrumented:
- HTTP endpoints
- Database operations
- External API calls
- Background tasks
"""

import os
import time
import random
import requests
import sqlite3
from flask import Flask, jsonify, request
from datetime import datetime
from opentelemetry.trace import Status, StatusCode

# Import OpenTelemetry setup
from otel import setup_instrumentation

app = Flask(__name__)

# Initialize OpenTelemetry instrumentation
logger, tracer, meter = setup_instrumentation(app, "flask-instrumentation-test")

# Create metrics instruments
request_counter = meter.create_counter(
    "http_requests_total",
    description="Total number of HTTP requests"
)
request_duration = meter.create_histogram(
    "http_request_duration_seconds",
    description="HTTP request duration in seconds"
)
db_operation_counter = meter.create_counter(
    "db_operations_total",
    description="Total number of database operations"
)
external_api_counter = meter.create_counter(
    "external_api_calls_total",
    description="Total number of external API calls"
)

# Initialize SQLite database


def init_db():
    """Initialize the SQLite database with a simple users table."""
    conn = sqlite3.connect('test.db')
    cursor = conn.cursor()
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            email TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    ''')

    # Insert some sample data
    cursor.execute("SELECT COUNT(*) FROM users")
    if cursor.fetchone()[0] == 0:
        sample_users = [
            ('Alice Johnson', 'alice@example.com'),
            ('Bob Smith', 'bob@example.com'),
            ('Charlie Brown', 'charlie@example.com')
        ]
        cursor.executemany('INSERT INTO users (name, email) VALUES (?, ?)', sample_users)

    conn.commit()
    conn.close()


def get_db_connection():
    """Get a database connection."""
    conn = sqlite3.connect('test.db')
    conn.row_factory = sqlite3.Row
    return conn


@app.route('/')
def home():
    """Home endpoint that returns basic application info."""
    start_time = time.time()

    logger.info("Home endpoint accessed")
    request_counter.add(1, {"method": "GET", "endpoint": "/"})

    response_data = {
        'message': 'Welcome to the OpenTelemetry Test Application',
        'version': '1.0.0',
        'timestamp': datetime.now().isoformat(),
        'endpoints': [
            '/users - Get all users',
            '/users/<id> - Get user by ID',
            '/users (POST) - Create new user',
            '/external - Make external API call',
            '/slow - Simulate slow operation',
            '/error - Simulate error condition'
        ]
    }

    # Record request duration
    duration = time.time() - start_time
    request_duration.record(duration, {"method": "GET", "endpoint": "/"})

    return jsonify(response_data)


@app.route('/users', methods=['GET'])
def get_users():
    """Get all users from the database."""
    start_time = time.time()

    with tracer.start_as_current_span("get_users_operation") as span:
        logger.info("Getting all users from database")
        request_counter.add(1, {"method": "GET", "endpoint": "/users"})
        db_operation_counter.add(1, {"operation": "select", "table": "users"})

        try:
            conn = get_db_connection()
            users = conn.execute('SELECT * FROM users ORDER BY created_at DESC').fetchall()
            conn.close()

            span.set_attribute("db.operation", "select")
            span.set_attribute("db.table", "users")
            span.set_attribute("users.count", len(users))

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/users"})

            return jsonify([dict(user) for user in users])
        except Exception as e:
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            logger.error(f"Error getting users: {e}")
            raise


@app.route('/users/<int:user_id>', methods=['GET'])
def get_user(user_id):
    """Get a specific user by ID."""
    start_time = time.time()

    with tracer.start_as_current_span("get_user_operation") as span:
        logger.info(f"Getting user with ID: {user_id}")
        request_counter.add(1, {"method": "GET", "endpoint": "/users/<id>"})
        db_operation_counter.add(1, {"operation": "select", "table": "users"})

        span.set_attribute("user.id", user_id)
        span.set_attribute("db.operation", "select")
        span.set_attribute("db.table", "users")

        try:
            conn = get_db_connection()
            user = conn.execute('SELECT * FROM users WHERE id = ?', (user_id,)).fetchone()
            conn.close()

            if user is None:
                span.set_attribute("user.found", False)
                logger.warning(f"User not found: {user_id}")
                # Record request duration
                duration = time.time() - start_time
                request_duration.record(duration, {"method": "GET", "endpoint": "/users/<id>", "status": "404"})
                return jsonify({'error': 'User not found'}), 404

            span.set_attribute("user.found", True)
            span.set_attribute("user.name", user['name'])

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/users/<id>", "status": "200"})

            return jsonify(dict(user))
        except Exception as e:
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            logger.error(f"Error getting user {user_id}: {e}")
            raise


@app.route('/users', methods=['POST'])
def create_user():
    """Create a new user."""
    start_time = time.time()

    with tracer.start_as_current_span("create_user_operation") as span:
        data = request.get_json()

        logger.info(f"Creating new user: {data.get('name') if data else 'unknown'}")
        request_counter.add(1, {"method": "POST", "endpoint": "/users"})

        if not data or 'name' not in data or 'email' not in data:
            span.set_attribute("validation.error", "missing_required_fields")
            logger.warning("User creation failed: missing name or email")
            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "POST", "endpoint": "/users", "status": "400"})
            return jsonify({'error': 'Name and email are required'}), 400

        span.set_attribute("user.name", data['name'])
        span.set_attribute("user.email", data['email'])
        span.set_attribute("db.operation", "insert")
        span.set_attribute("db.table", "users")

        try:
            db_operation_counter.add(1, {"operation": "insert", "table": "users"})

            conn = get_db_connection()
            cursor = conn.cursor()
            cursor.execute(
                'INSERT INTO users (name, email) VALUES (?, ?)',
                (data['name'], data['email'])
            )
            user_id = cursor.lastrowid
            conn.commit()
            conn.close()

            span.set_attribute("user.id", user_id)
            logger.info(f"User created successfully with ID: {user_id}")

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "POST", "endpoint": "/users", "status": "201"})

            return jsonify({'id': user_id, 'message': 'User created successfully'}), 201
        except Exception as e:
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            logger.error(f"Error creating user: {e}")
            raise


@app.route('/external')
def external_call():
    """Make an external API call to demonstrate external service instrumentation."""
    start_time = time.time()

    with tracer.start_as_current_span("external_api_call") as span:
        logger.info("Making external API call to JSONPlaceholder")
        request_counter.add(1, {"method": "GET", "endpoint": "/external"})
        external_api_counter.add(1, {"service": "jsonplaceholder", "operation": "get_post"})

        span.set_attribute("external.service", "jsonplaceholder.typicode.com")
        span.set_attribute("external.operation", "get_post")
        span.set_attribute("external.url", "https://jsonplaceholder.typicode.com/posts/1")

        try:
            # Call a public API (JSONPlaceholder)
            response = requests.get('https://jsonplaceholder.typicode.com/posts/1', timeout=5)
            response.raise_for_status()

            span.set_attribute("http.status_code", response.status_code)
            span.set_attribute("external.success", True)
            logger.info(f"External API call successful, status: {response.status_code}")

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/external", "status": "200"})

            return jsonify({
                'message': 'External API call successful',
                'data': response.json(),
                'status_code': response.status_code
            })
        except requests.RequestException as e:
            span.record_exception(e)
            span.set_status(Status(StatusCode.ERROR, str(e)))
            span.set_attribute("external.success", False)
            logger.error(f"External API call failed: {e}")

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/external", "status": "500"})

            return jsonify({
                'error': 'External API call failed',
                'details': str(e)
            }), 500


@app.route('/slow')
def slow_operation():
    """Simulate a slow operation to demonstrate latency tracking."""
    start_time = time.time()

    with tracer.start_as_current_span("slow_operation") as span:
        # Random delay between 1-3 seconds
        delay = random.uniform(1.0, 3.0)

        logger.info(f"Starting slow operation with {delay:.2f}s delay")
        request_counter.add(1, {"method": "GET", "endpoint": "/slow"})

        span.set_attribute("operation.delay_seconds", delay)
        span.set_attribute("operation.type", "simulated_slow")

        time.sleep(delay)

        logger.info(f"Slow operation completed after {delay:.2f}s")

        # Record request duration
        duration = time.time() - start_time
        request_duration.record(duration, {"method": "GET", "endpoint": "/slow"})

        return jsonify({
            'message': 'Slow operation completed',
            'delay_seconds': round(delay, 2),
            'timestamp': datetime.now().isoformat()
        })


@app.route('/error')
def error_endpoint():
    """Simulate an error condition to demonstrate error tracking."""
    start_time = time.time()
    error_type = request.args.get('type', 'generic')

    with tracer.start_as_current_span("error_simulation") as span:
        logger.info(f"Simulating error type: {error_type}")
        request_counter.add(1, {"method": "GET", "endpoint": "/error"})

        span.set_attribute("error.type", error_type)
        span.set_attribute("error.simulated", True)

        if error_type == 'db':
            # Simulate database error
            span.set_attribute("error.category", "database")
            logger.error("Simulating database error")

            conn = get_db_connection()
            try:
                conn.execute('SELECT * FROM non_existent_table')
            except sqlite3.OperationalError as e:
                span.record_exception(e)
                span.set_status(Status(StatusCode.ERROR, str(e)))
                conn.close()

                # Record request duration
                duration = time.time() - start_time
                request_duration.record(duration, {"method": "GET", "endpoint": "/error", "status": "500"})

                return jsonify({'error': 'Database error', 'details': str(e)}), 500
            finally:
                conn.close()
        elif error_type == 'timeout':
            # Simulate timeout
            span.set_attribute("error.category", "timeout")
            logger.error("Simulating timeout error")

            time.sleep(0.1)
            span.set_status(Status(StatusCode.ERROR, "Operation timed out"))

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/error", "status": "408"})

            return jsonify({'error': 'Operation timed out'}), 408
        elif error_type == 'auth':
            # Simulate authentication error
            span.set_attribute("error.category", "authentication")
            logger.error("Simulating authentication error")
            span.set_status(Status(StatusCode.ERROR, "Unauthorized access"))

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/error", "status": "401"})

            return jsonify({'error': 'Unauthorized access'}), 401
        else:
            # Generic server error
            span.set_attribute("error.category", "generic")
            logger.error("Simulating generic server error")
            span.set_status(Status(StatusCode.ERROR, "Internal server error"))

            # Record request duration
            duration = time.time() - start_time
            request_duration.record(duration, {"method": "GET", "endpoint": "/error", "status": "500"})

            return jsonify({'error': 'Internal server error'}), 500


@app.route('/health')
def health_check():
    """Health check endpoint."""
    start_time = time.time()

    logger.info("Health check endpoint accessed")
    request_counter.add(1, {"method": "GET", "endpoint": "/health"})

    response_data = {
        'status': 'healthy',
        'timestamp': datetime.now().isoformat(),
        'uptime': 'N/A'  # Could be calculated if we track start time
    }

    # Record request duration
    duration = time.time() - start_time
    request_duration.record(duration, {"method": "GET", "endpoint": "/health"})

    return jsonify(response_data)


if __name__ == '__main__':
    # Initialize database
    init_db()

    logger.info("Starting Flask application with OpenTelemetry instrumentation")
    logger.info(f"OTEL endpoint: {os.environ.get('OTEL_EXPORTER_OTLP_ENDPOINT', 'http://localhost:4317')}")

    # Run the application
    app.run(host='0.0.0.0', port=5000, debug=True)
