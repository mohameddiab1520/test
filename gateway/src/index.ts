import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import cors from 'cors';
import helmet from 'helmet';
import rateLimit from 'express-rate-limit';
import { config, validateConfig } from './config';
import { authenticateToken, optionalAuth } from './middleware/auth';
import { requestLogger, errorLogger } from './middleware/logger';
import { errorHandler, notFoundHandler } from './middleware/errorHandler';

// Validate configuration
validateConfig();

const app = express();

// Security middleware
app.use(helmet());

// CORS configuration
app.use(cors({
  origin: config.cors.origin,
  credentials: config.cors.credentials
}));

// Body parsing
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ extended: true, limit: '50mb' }));

// Request logging
app.use(requestLogger);

// Rate limiting
const limiter = rateLimit({
  windowMs: config.rateLimit.windowMs,
  max: config.rateLimit.max,
  message: {
    error: 'Too Many Requests',
    message: 'Too many requests from this IP, please try again later.'
  },
  standardHeaders: true,
  legacyHeaders: false
});

app.use('/api/', limiter);

// Health check
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'gateway',
    version: '1.0.0',
    timestamp: new Date().toISOString()
  });
});

// Metrics endpoint
app.get('/metrics', (req, res) => {
  res.json({
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    timestamp: new Date().toISOString()
  });
});

// Auth Service Routes (public - no auth required)
app.use('/api/v1/auth', createProxyMiddleware({
  target: config.services.auth.url,
  changeOrigin: true,
  timeout: config.services.auth.timeout,
  onProxyReq: (proxyReq, req, res) => {
    console.log(`[AUTH] ${req.method} ${req.path}`);
  },
  onError: (err, req, res) => {
    console.error('[AUTH] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Auth service is currently unavailable'
    });
  }
}));

// Session Service Routes (requires authentication)
app.use('/api/v1/sessions', authenticateToken, createProxyMiddleware({
  target: config.services.session.url,
  changeOrigin: true,
  timeout: config.services.session.timeout,
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[SESSION] ${req.method} ${req.path}`);

    // Forward user info in headers
    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[SESSION] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Session service is currently unavailable'
    });
  }
}));

// Asset Service Routes (requires authentication)
app.use('/api/v1/assets', authenticateToken, createProxyMiddleware({
  target: config.services.asset.url,
  changeOrigin: true,
  timeout: config.services.asset.timeout,
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[ASSET] ${req.method} ${req.path}`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[ASSET] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Asset service is currently unavailable'
    });
  }
}));

// Presence Service Routes (requires authentication)
app.use('/api/v1/presence', authenticateToken, createProxyMiddleware({
  target: config.services.presence.url,
  changeOrigin: true,
  timeout: config.services.presence.timeout,
  ws: true, // Enable WebSocket support
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[PRESENCE] ${req.method} ${req.path}`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[PRESENCE] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Presence service is currently unavailable'
    });
  }
}));

// Voice Service Routes (requires authentication)
app.use('/api/v1/voice', authenticateToken, createProxyMiddleware({
  target: config.services.voice.url,
  changeOrigin: true,
  timeout: config.services.voice.timeout,
  ws: true, // Enable WebSocket support
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[VOICE] ${req.method} ${req.path}`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[VOICE] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Voice service is currently unavailable'
    });
  }
}));

// Analytics Service Routes (requires authentication)
app.use('/api/v1/analytics', authenticateToken, createProxyMiddleware({
  target: config.services.analytics.url,
  changeOrigin: true,
  timeout: config.services.analytics.timeout,
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[ANALYTICS] ${req.method} ${req.path}`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[ANALYTICS] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Analytics service is currently unavailable'
    });
  }
}));

// Build Service Routes (requires authentication)
app.use('/api/v1/builds', authenticateToken, createProxyMiddleware({
  target: config.services.build.url,
  changeOrigin: true,
  timeout: config.services.build.timeout,
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[BUILD] ${req.method} ${req.path}`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[BUILD] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Build service is currently unavailable'
    });
  }
}));

// Conflict Service Routes (requires authentication)
app.use('/api/v1/conflicts', authenticateToken, createProxyMiddleware({
  target: config.services.conflict.url,
  changeOrigin: true,
  timeout: config.services.conflict.timeout,
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[CONFLICT] ${req.method} ${req.path}`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[CONFLICT] Proxy error:', err);
    res.status(503).json({
      error: 'Service Unavailable',
      message: 'Conflict service is currently unavailable'
    });
  }
}));

// WebSocket proxy to Sync Service (requires authentication)
app.use('/ws', authenticateToken, createProxyMiddleware({
  target: config.services.sync.url,
  ws: true,
  changeOrigin: true,
  onProxyReq: (proxyReq, req: any, res) => {
    console.log(`[SYNC] WebSocket connection`);

    if (req.user) {
      proxyReq.setHeader('X-User-Id', req.user.id);
      proxyReq.setHeader('X-User-Email', req.user.email);
      proxyReq.setHeader('X-User-Role', req.user.role);
    }
  },
  onError: (err, req, res) => {
    console.error('[SYNC] Proxy error:', err);
  }
}));

// Error logging
app.use(errorLogger);

// 404 handler
app.use(notFoundHandler);

// Global error handler
app.use(errorHandler);

// Start server
const PORT = config.port;

app.listen(PORT, () => {
  console.log(`API Gateway listening on port ${PORT}`);
  console.log(`Routes:`);
  console.log(`  Health: http://localhost:${PORT}/health`);
  console.log(`  Auth: http://localhost:${PORT}/api/v1/auth/*`);
  console.log(`  Sessions: http://localhost:${PORT}/api/v1/sessions/*`);
  console.log(`  Assets: http://localhost:${PORT}/api/v1/assets/*`);
  console.log(`  Presence: http://localhost:${PORT}/api/v1/presence/*`);
  console.log(`  Voice: http://localhost:${PORT}/api/v1/voice/*`);
  console.log(`  Analytics: http://localhost:${PORT}/api/v1/analytics/*`);
  console.log(`  Builds: http://localhost:${PORT}/api/v1/builds/*`);
  console.log(`  Conflicts: http://localhost:${PORT}/api/v1/conflicts/*`);
  console.log(`  WebSocket: ws://localhost:${PORT}/ws`);
});

// Graceful shutdown
const shutdown = (signal: string) => {
  console.log(`${signal} received. Closing server...`);
  process.exit(0);
};

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));
