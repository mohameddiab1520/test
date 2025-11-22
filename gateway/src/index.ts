import express from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import cors from 'cors';
import helmet from 'helmet';
import rateLimit from 'express-rate-limit';

const app = express();
const PORT = process.env.PORT || 3000;

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100, // limit each IP to 100 requests per windowMs
  message: 'Too many requests from this IP, please try again later.'
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

// Service routes
const SESSION_SERVICE = process.env.SESSION_SERVICE_URL || 'http://localhost:8080';
const SYNC_SERVICE = process.env.SYNC_SERVICE_URL || 'http://localhost:8081';
const ASSET_SERVICE = process.env.ASSET_SERVICE_URL || 'http://localhost:8082';
const AUTH_SERVICE = process.env.AUTH_SERVICE_URL || 'http://localhost:8083';

// Proxy to Auth Service
app.use('/api/v1/auth', createProxyMiddleware({
  target: AUTH_SERVICE,
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log(`[AUTH] ${req.method} ${req.path}`);
  }
}));

// Proxy to Session Service
app.use('/api/v1/sessions', createProxyMiddleware({
  target: SESSION_SERVICE,
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log(`[SESSION] ${req.method} ${req.path}`);
  }
}));

// Proxy to Asset Service
app.use('/api/v1/assets', createProxyMiddleware({
  target: ASSET_SERVICE,
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log(`[ASSET] ${req.method} ${req.path}`);
  }
}));

// WebSocket proxy to Sync Service
app.use('/ws', createProxyMiddleware({
  target: SYNC_SERVICE,
  ws: true,
  changeOrigin: true,
  onProxyReq: (proxyReq, req, res) => {
    console.log(`[SYNC] WebSocket connection`);
  }
}));

// 404 handler
app.use((req, res) => {
  res.status(404).json({
    error: 'Not Found',
    message: `Route ${req.method} ${req.path} not found`
  });
});

// Error handler
app.use((err: any, req: express.Request, res: express.Response, next: express.NextFunction) => {
  console.error('Error:', err);
  res.status(500).json({
    error: 'Internal Server Error',
    message: err.message
  });
});

// Start server
app.listen(PORT, () => {
  console.log(`API Gateway listening on port ${PORT}`);
  console.log(`Routes:`);
  console.log(`  Health: http://localhost:${PORT}/health`);
  console.log(`  Auth: http://localhost:${PORT}/api/v1/auth/*`);
  console.log(`  Sessions: http://localhost:${PORT}/api/v1/sessions/*`);
  console.log(`  Assets: http://localhost:${PORT}/api/v1/assets/*`);
  console.log(`  WebSocket: ws://localhost:${PORT}/ws`);
});
