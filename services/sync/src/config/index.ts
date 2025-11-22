import { SyncConfig } from '../types';
import dotenv from 'dotenv';

dotenv.config();

export const config: SyncConfig = {
  port: parseInt(process.env.PORT || '8081', 10),

  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: parseInt(process.env.REDIS_PORT || '6379', 10),
    password: process.env.REDIS_PASSWORD,
    db: parseInt(process.env.REDIS_DB || '0', 10)
  },

  jwtSecret: process.env.JWT_SECRET || 'your-secret-key-change-in-production',

  logLevel: process.env.LOG_LEVEL || 'info',

  // Keep last 1000 operations per session in memory
  maxOperationsHistory: parseInt(process.env.MAX_OPERATIONS_HISTORY || '1000', 10),

  // Operation timeout in milliseconds (30 seconds)
  operationTimeout: parseInt(process.env.OPERATION_TIMEOUT || '30000', 10),

  // Heartbeat interval in milliseconds (30 seconds)
  heartbeatInterval: parseInt(process.env.HEARTBEAT_INTERVAL || '30000', 10)
};

export function validateConfig(): void {
  if (!config.jwtSecret || config.jwtSecret === 'your-secret-key-change-in-production') {
    console.warn('WARNING: Using default JWT secret. Set JWT_SECRET environment variable in production!');
  }

  if (!config.redis.host) {
    throw new Error('REDIS_HOST is required');
  }

  console.log('Configuration loaded:');
  console.log(`  Port: ${config.port}`);
  console.log(`  Redis: ${config.redis.host}:${config.redis.port}`);
  console.log(`  Log Level: ${config.logLevel}`);
}
