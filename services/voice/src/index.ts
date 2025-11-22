import express from 'express';
import { createServer } from 'http';
import { Server as SocketIOServer } from 'socket.io';
import cors from 'cors';
import dotenv from 'dotenv';
import { config } from './config';
import { VoiceService } from './services/VoiceService';
import { MetricsService } from './services/MetricsService';
import { setupRoutes } from './routes';
import { setupSocketHandlers } from './socket';
import { logger } from './utils/logger';

dotenv.config();

async function main() {
  const app = express();
  const httpServer = createServer(app);

  // Socket.IO server
  const io = new SocketIOServer(httpServer, {
    cors: {
      origin: '*',
      methods: ['GET', 'POST']
    },
    transports: ['websocket', 'polling']
  });

  // Middleware
  app.use(cors());
  app.use(express.json());

  // Health check
  app.get('/health', (req, res) => {
    res.json({
      status: 'healthy',
      service: 'voice-service',
      timestamp: new Date().toISOString()
    });
  });

  // Initialize services
  const metricsService = new MetricsService();
  const voiceService = new VoiceService(metricsService);

  await voiceService.initialize();

  // Setup routes
  setupRoutes(app, voiceService, metricsService);

  // Setup Socket.IO handlers
  setupSocketHandlers(io, voiceService);

  // Start server
  const port = config.servicePort;
  httpServer.listen(port, () => {
    logger.info(`Voice Service started on port ${port}`);
  });

  // Graceful shutdown
  process.on('SIGTERM', async () => {
    logger.info('SIGTERM received, shutting down gracefully...');
    await voiceService.shutdown();
    httpServer.close(() => {
      logger.info('Server closed');
      process.exit(0);
    });
  });
}

main().catch((error) => {
  logger.error('Failed to start Voice Service', { error });
  process.exit(1);
});
