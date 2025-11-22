import { WebSocketServer, WebSocket } from 'ws';
import express from 'express';
import { createServer } from 'http';
import { config, validateConfig } from './config';
import { SessionManager } from './session/SessionManager';
import { getRedisClient, closeRedisClient } from './redis/RedisClient';
import { logger } from './utils/logger';

// Validate configuration
validateConfig();

const app = express();
const server = createServer(app);
const wss = new WebSocketServer({ server });
const sessionManager = new SessionManager();

// Middleware
app.use(express.json());

// Health check endpoint
app.get('/health', async (req, res) => {
  try {
    const redis = getRedisClient();

    res.json({
      status: 'healthy',
      service: 'sync',
      version: '1.0.0',
      connections: sessionManager.getConnectionCount(),
      sessions: sessionManager.getSessionCount(),
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      status: 'unhealthy',
      error: error instanceof Error ? error.message : 'Unknown error'
    });
  }
});

// Metrics endpoint
app.get('/metrics', (req, res) => {
  res.json({
    connections: sessionManager.getConnectionCount(),
    sessions: sessionManager.getSessionCount(),
    uptime: process.uptime(),
    memory: process.memoryUsage()
  });
});

// WebSocket connection handler
wss.on('connection', async (ws: WebSocket, req) => {
  try {
    const clientId = await sessionManager.handleConnection(ws);
    logger.info('Client connected', { clientId, ip: req.socket.remoteAddress });
  } catch (error) {
    logger.error('Error handling connection', { error });
    ws.close();
  }
});

// Start server
server.listen(config.port, () => {
  logger.info('Sync Service started', {
    port: config.port,
    wsEndpoint: `ws://localhost:${config.port}`,
    healthCheck: `http://localhost:${config.port}/health`
  });
});

// Graceful shutdown
const shutdown = async (signal: string) => {
  logger.info(`${signal} signal received: closing server`);

  // Stop accepting new connections
  wss.close(() => {
    logger.info('WebSocket server closed');
  });

  server.close(() => {
    logger.info('HTTP server closed');
  });

  try {
    // Cleanup session manager
    await sessionManager.shutdown();

    // Close Redis connections
    await closeRedisClient();

    logger.info('Graceful shutdown complete');
    process.exit(0);
  } catch (error) {
    logger.error('Error during shutdown', { error });
    process.exit(1);
  }
};

process.on('SIGTERM', () => shutdown('SIGTERM'));
process.on('SIGINT', () => shutdown('SIGINT'));

// Handle uncaught errors
process.on('uncaughtException', (error) => {
  logger.error('Uncaught exception', { error });
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  logger.error('Unhandled rejection', { reason, promise });
  process.exit(1);
});
