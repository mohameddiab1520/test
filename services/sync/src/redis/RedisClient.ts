import Redis from 'ioredis';
import { config } from '../config';
import { SyncOperation, SessionState } from '../types';

export class RedisClient {
  private client: Redis;
  private subscriber: Redis;
  private publisher: Redis;

  constructor() {
    const redisConfig = {
      host: config.redis.host,
      port: config.redis.port,
      password: config.redis.password,
      db: config.redis.db,
      retryStrategy: (times: number) => {
        const delay = Math.min(times * 50, 2000);
        return delay;
      },
      maxRetriesPerRequest: 3
    };

    this.client = new Redis(redisConfig);
    this.subscriber = new Redis(redisConfig);
    this.publisher = new Redis(redisConfig);

    this.setupEventHandlers();
  }

  private setupEventHandlers(): void {
    this.client.on('connect', () => {
      console.log('Redis client connected');
    });

    this.client.on('error', (err) => {
      console.error('Redis client error:', err);
    });

    this.subscriber.on('connect', () => {
      console.log('Redis subscriber connected');
    });

    this.publisher.on('connect', () => {
      console.log('Redis publisher connected');
    });
  }

  // Session operations
  async getSessionState(sessionId: string): Promise<SessionState | null> {
    const key = `session:${sessionId}`;
    const data = await this.client.get(key);

    if (!data) {
      return null;
    }

    const parsed = JSON.parse(data);
    return {
      ...parsed,
      clients: new Set(parsed.clients),
      lastUpdated: new Date(parsed.lastUpdated)
    };
  }

  async setSessionState(state: SessionState): Promise<void> {
    const key = `session:${state.sessionId}`;
    const data = JSON.stringify({
      ...state,
      clients: Array.from(state.clients),
      lastUpdated: state.lastUpdated.toISOString()
    });

    // Set with 24 hour expiry
    await this.client.setex(key, 86400, data);
  }

  async deleteSession(sessionId: string): Promise<void> {
    const key = `session:${sessionId}`;
    await this.client.del(key);
  }

  // Operation history
  async addOperation(sessionId: string, operation: SyncOperation): Promise<void> {
    const key = `operations:${sessionId}`;
    const data = JSON.stringify(operation);

    // Add to list and trim to max size
    await this.client.rpush(key, data);
    await this.client.ltrim(key, -config.maxOperationsHistory, -1);

    // Set expiry
    await this.client.expire(key, 86400);
  }

  async getOperations(sessionId: string, limit: number = 100): Promise<SyncOperation[]> {
    const key = `operations:${sessionId}`;
    const operations = await this.client.lrange(key, -limit, -1);

    return operations.map(op => {
      const parsed = JSON.parse(op);
      return {
        ...parsed,
        timestamp: new Date(parsed.timestamp)
      };
    });
  }

  async getOperationsSince(sessionId: string, version: number): Promise<SyncOperation[]> {
    const operations = await this.getOperations(sessionId, config.maxOperationsHistory);
    return operations.filter(op => op.version > version);
  }

  // Pub/Sub for cross-instance communication
  async publishOperation(sessionId: string, operation: SyncOperation): Promise<void> {
    const channel = `sync:${sessionId}`;
    const data = JSON.stringify(operation);
    await this.publisher.publish(channel, data);
  }

  async subscribeToSession(sessionId: string, callback: (operation: SyncOperation) => void): Promise<void> {
    const channel = `sync:${sessionId}`;

    await this.subscriber.subscribe(channel);

    this.subscriber.on('message', (ch, message) => {
      if (ch === channel) {
        try {
          const operation = JSON.parse(message);
          operation.timestamp = new Date(operation.timestamp);
          callback(operation);
        } catch (error) {
          console.error('Error parsing published operation:', error);
        }
      }
    });
  }

  async unsubscribeFromSession(sessionId: string): Promise<void> {
    const channel = `sync:${sessionId}`;
    await this.subscriber.unsubscribe(channel);
  }

  // Client presence
  async addClientToSession(sessionId: string, clientId: string): Promise<void> {
    const key = `session:clients:${sessionId}`;
    await this.client.sadd(key, clientId);
    await this.client.expire(key, 86400);
  }

  async removeClientFromSession(sessionId: string, clientId: string): Promise<void> {
    const key = `session:clients:${sessionId}`;
    await this.client.srem(key, clientId);
  }

  async getSessionClients(sessionId: string): Promise<string[]> {
    const key = `session:clients:${sessionId}`;
    return await this.client.smembers(key);
  }

  async getClientCount(sessionId: string): Promise<number> {
    const key = `session:clients:${sessionId}`;
    return await this.client.scard(key);
  }

  // Client heartbeat
  async updateClientHeartbeat(clientId: string, sessionId: string): Promise<void> {
    const key = `heartbeat:${clientId}`;
    await this.client.setex(key, 60, sessionId); // 60 second TTL
  }

  async isClientAlive(clientId: string): Promise<boolean> {
    const key = `heartbeat:${clientId}`;
    const exists = await this.client.exists(key);
    return exists === 1;
  }

  // Cleanup
  async disconnect(): Promise<void> {
    await Promise.all([
      this.client.quit(),
      this.subscriber.quit(),
      this.publisher.quit()
    ]);
    console.log('Redis connections closed');
  }
}

// Singleton instance
let redisClient: RedisClient | null = null;

export function getRedisClient(): RedisClient {
  if (!redisClient) {
    redisClient = new RedisClient();
  }
  return redisClient;
}

export async function closeRedisClient(): Promise<void> {
  if (redisClient) {
    await redisClient.disconnect();
    redisClient = null;
  }
}
