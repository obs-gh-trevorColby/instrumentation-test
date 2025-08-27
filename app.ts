// CRITICAL: Initialize OpenTelemetry FIRST before any other imports
import { initOtel, shutdownOtel, tracer, logger, meter } from './otel';

// Initialize OpenTelemetry
const { tracer: otelTracer, logger: otelLogger, meter: otelMeter } = initOtel();

// Now import other modules after OpenTelemetry initialization for auto-instrumentation
import express, { Request, Response, NextFunction } from 'express';
import axios from 'axios';
import { trace, context, SpanStatusCode } from '@opentelemetry/api';
import { SeverityNumber } from '@opentelemetry/api-logs';

const app = express();
const PORT = process.env.PORT || 3000;

// Initialize metrics
const requestCounter = otelMeter.createCounter('http_requests_total', {
  description: 'Total number of HTTP requests',
});

const requestDuration = otelMeter.createHistogram('http_request_duration_ms', {
  description: 'Duration of HTTP requests in milliseconds',
});

const userOperationsCounter = otelMeter.createCounter('user_operations_total', {
  description: 'Total number of user operations',
});

// Middleware
app.use(express.json());

// Request logging middleware
app.use((req: Request, res: Response, next: NextFunction) => {
  const startTime = Date.now();
  
  // Log request start
  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'HTTP request started',
    attributes: {
      method: req.method,
      url: req.url,
      userAgent: req.get('User-Agent') || 'unknown',
    },
  });

  // Override res.end to capture response metrics
  const originalEnd = res.end.bind(res);
  res.end = function(chunk?: any, encoding?: BufferEncoding | (() => void), cb?: () => void) {
    const duration = Date.now() - startTime;

    // Record metrics
    requestCounter.add(1, {
      method: req.method,
      route: req.route?.path || req.path,
      status_code: res.statusCode.toString(),
    });

    requestDuration.record(duration, {
      method: req.method,
      route: req.route?.path || req.path,
      status_code: res.statusCode.toString(),
    });

    // Log request completion
    otelLogger.emit({
      severityNumber: SeverityNumber.INFO,
      severityText: 'INFO',
      body: 'HTTP request completed',
      attributes: {
        method: req.method,
        url: req.url,
        statusCode: res.statusCode,
        duration: duration,
      },
    });

    return originalEnd(chunk, encoding as BufferEncoding, cb);
  };

  next();
});

// In-memory data store
let users = [
  { id: 1, name: 'John Doe', email: 'john@example.com' },
  { id: 2, name: 'Jane Smith', email: 'jane@example.com' }
];

// Routes
app.get('/', (req: Request, res: Response) => {
  const span = trace.getActiveSpan();
  if (span) {
    span.setAttributes({
      'app.route': '/',
      'app.handler': 'root',
    });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Root endpoint accessed',
  });

  res.json({ message: 'Welcome to the Instrumentation Test API' });
});

app.get('/health', (req: Request, res: Response) => {
  const span = trace.getActiveSpan();
  if (span) {
    span.setAttributes({
      'app.route': '/health',
      'app.handler': 'health_check',
    });
  }

  const healthData = { 
    status: 'healthy', 
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
  };

  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Health check performed',
    attributes: healthData,
  });

  res.json(healthData);
});

app.get('/users', (req: Request, res: Response) => {
  const span = trace.getActiveSpan();
  if (span) {
    span.setAttributes({
      'app.route': '/users',
      'app.handler': 'get_all_users',
      'users.count': users.length,
    });
  }

  userOperationsCounter.add(1, { operation: 'list' });

  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Fetching all users',
    attributes: { userCount: users.length },
  });

  res.json(users);
});

app.get('/users/:id', (req: Request, res: Response) => {
  const userId = parseInt(req.params.id);
  const span = trace.getActiveSpan();
  
  if (span) {
    span.setAttributes({
      'app.route': '/users/:id',
      'app.handler': 'get_user_by_id',
      'user.id': userId,
    });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Fetching user by ID',
    attributes: { userId },
  });
  
  const user = users.find(u => u.id === userId);
  if (!user) {
    if (span) {
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'User not found' });
      span.setAttributes({ 'error.type': 'not_found' });
    }

    userOperationsCounter.add(1, { operation: 'get', result: 'not_found' });

    otelLogger.emit({
      severityNumber: SeverityNumber.WARN,
      severityText: 'WARN',
      body: 'User not found',
      attributes: { userId },
    });

    return res.status(404).json({ error: 'User not found' });
  }

  userOperationsCounter.add(1, { operation: 'get', result: 'success' });
  
  if (span) {
    span.setAttributes({
      'user.name': user.name,
      'user.email': user.email,
    });
  }

  res.json(user);
});

app.post('/users', (req: Request, res: Response) => {
  const { name, email } = req.body;
  const span = trace.getActiveSpan();

  if (span) {
    span.setAttributes({
      'app.route': '/users',
      'app.handler': 'create_user',
      'user.name': name,
      'user.email': email,
    });
  }

  if (!name || !email) {
    if (span) {
      span.setStatus({ code: SpanStatusCode.ERROR, message: 'Missing required fields' });
      span.setAttributes({ 'error.type': 'validation_error' });
    }

    userOperationsCounter.add(1, { operation: 'create', result: 'validation_error' });

    otelLogger.emit({
      severityNumber: SeverityNumber.WARN,
      severityText: 'WARN',
      body: 'User creation failed - missing required fields',
      attributes: { providedName: !!name, providedEmail: !!email },
    });

    return res.status(400).json({ error: 'Name and email are required' });
  }

  const newUser = {
    id: users.length + 1,
    name,
    email
  };

  users.push(newUser);
  userOperationsCounter.add(1, { operation: 'create', result: 'success' });

  if (span) {
    span.setAttributes({
      'user.id': newUser.id,
      'users.total_count': users.length,
    });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'New user created successfully',
    attributes: {
      userId: newUser.id,
      userName: newUser.name,
      userEmail: newUser.email,
      totalUsers: users.length,
    },
  });

  res.status(201).json(newUser);
});

app.get('/external-api', async (req: Request, res: Response) => {
  const span = trace.getActiveSpan();

  if (span) {
    span.setAttributes({
      'app.route': '/external-api',
      'app.handler': 'external_api_call',
    });
  }

  try {
    otelLogger.emit({
      severityNumber: SeverityNumber.INFO,
      severityText: 'INFO',
      body: 'Making external API call',
      attributes: { target: 'jsonplaceholder.typicode.com' },
    });

    const response = await axios.get('https://jsonplaceholder.typicode.com/posts/1');

    if (span) {
      span.setAttributes({
        'http.external.status_code': response.status,
        'http.external.response_size': JSON.stringify(response.data).length,
      });
    }

    otelLogger.emit({
      severityNumber: SeverityNumber.INFO,
      severityText: 'INFO',
      body: 'External API call successful',
      attributes: {
        statusCode: response.status,
        responseSize: JSON.stringify(response.data).length,
      },
    });

    res.json(response.data);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';

    if (span) {
      span.setStatus({ code: SpanStatusCode.ERROR, message: errorMessage });
      span.setAttributes({
        'error.type': 'external_api_error',
        'error.message': errorMessage,
      });
    }

    otelLogger.emit({
      severityNumber: SeverityNumber.ERROR,
      severityText: 'ERROR',
      body: 'External API call failed',
      attributes: {
        error: errorMessage,
        target: 'jsonplaceholder.typicode.com',
      },
    });

    res.status(500).json({ error: 'Failed to fetch external data' });
  }
});

app.get('/slow', (req: Request, res: Response) => {
  const span = trace.getActiveSpan();

  if (span) {
    span.setAttributes({
      'app.route': '/slow',
      'app.handler': 'slow_operation',
      'operation.duration_ms': 2000,
    });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Processing slow request',
    attributes: { expectedDuration: 2000 },
  });

  setTimeout(() => {
    otelLogger.emit({
      severityNumber: SeverityNumber.INFO,
      severityText: 'INFO',
      body: 'Slow operation completed',
    });

    res.json({ message: 'This was a slow operation', duration: '2 seconds' });
  }, 2000);
});

app.get('/error', (req: Request, res: Response) => {
  const span = trace.getActiveSpan();

  if (span) {
    span.setAttributes({
      'app.route': '/error',
      'app.handler': 'intentional_error',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: 'Intentional error for testing' });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.ERROR,
    severityText: 'ERROR',
    body: 'Intentional error triggered for testing',
    attributes: { intentional: true },
  });

  res.status(500).json({ error: 'This is an intentional error for testing' });
});

// Error handling middleware
app.use((err: Error, req: Request, res: Response, next: NextFunction) => {
  const span = trace.getActiveSpan();

  if (span) {
    span.setStatus({ code: SpanStatusCode.ERROR, message: err.message });
    span.setAttributes({
      'error.type': 'unhandled_error',
      'error.message': err.message,
      'error.stack': err.stack || 'No stack trace available',
    });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.ERROR,
    severityText: 'ERROR',
    body: 'Unhandled error occurred',
    attributes: {
      error: err.message,
      stack: err.stack,
      url: req.url,
      method: req.method,
    },
  });

  res.status(500).json({ error: 'Internal server error' });
});

// 404 handler
app.use((req: Request, res: Response) => {
  const span = trace.getActiveSpan();

  if (span) {
    span.setAttributes({
      'app.route': 'not_found',
      'app.handler': '404_handler',
    });
    span.setStatus({ code: SpanStatusCode.ERROR, message: 'Route not found' });
  }

  otelLogger.emit({
    severityNumber: SeverityNumber.WARN,
    severityText: 'WARN',
    body: 'Route not found',
    attributes: {
      url: req.url,
      method: req.method,
    },
  });

  res.status(404).json({ error: 'Route not found' });
});

const server = app.listen(PORT, () => {
  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Server started successfully',
    attributes: {
      port: PORT,
      environment: process.env.NODE_ENV || 'development',
    },
  });

  console.log(`Server is running on port ${PORT}`);
});

// Graceful shutdown
const gracefulShutdown = () => {
  otelLogger.emit({
    severityNumber: SeverityNumber.INFO,
    severityText: 'INFO',
    body: 'Graceful shutdown initiated',
  });

  server.close(() => {
    otelLogger.emit({
      severityNumber: SeverityNumber.INFO,
      severityText: 'INFO',
      body: 'Server closed successfully',
    });

    shutdownOtel();
    process.exit(0);
  });
};

process.on('SIGTERM', gracefulShutdown);
process.on('SIGINT', gracefulShutdown);

export default app;
