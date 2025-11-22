import dotenv from 'dotenv';

dotenv.config();

export interface ServiceConfig {
  url: string;
  timeout: number;
}

export interface GatewayConfig {
  port: number;
  services: {
    auth: ServiceConfig;
    session: ServiceConfig;
    sync: ServiceConfig;
    asset: ServiceConfig;
    presence: ServiceConfig;
    voice: ServiceConfig;
    analytics: ServiceConfig;
  };
  rateLimit: {
    windowMs: number;
    max: number;
  };
  cors: {
    origin: string | string[];
    credentials: boolean;
  };
  jwt: {
    secret: string;
  };
}

export const config: GatewayConfig = {
  port: parseInt(process.env.PORT || '3000', 10),

  services: {
    auth: {
      url: process.env.AUTH_SERVICE_URL || 'http://localhost:8083',
      timeout: 30000
    },
    session: {
      url: process.env.SESSION_SERVICE_URL || 'http://localhost:8080',
      timeout: 30000
    },
    sync: {
      url: process.env.SYNC_SERVICE_URL || 'http://localhost:8081',
      timeout: 30000
    },
    asset: {
      url: process.env.ASSET_SERVICE_URL || 'http://localhost:8082',
      timeout: 60000 // Longer timeout for file uploads
    },
    presence: {
      url: process.env.PRESENCE_SERVICE_URL || 'http://localhost:8084',
      timeout: 30000
    },
    voice: {
      url: process.env.VOICE_SERVICE_URL || 'http://localhost:8085',
      timeout: 30000
    },
    analytics: {
      url: process.env.ANALYTICS_SERVICE_URL || 'http://localhost:8086',
      timeout: 30000
    }
  },

  rateLimit: {
    windowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS || '900000', 10), // 15 minutes
    max: parseInt(process.env.RATE_LIMIT_MAX || '100', 10)
  },

  cors: {
    origin: process.env.CORS_ORIGIN?.split(',') || '*',
    credentials: process.env.CORS_CREDENTIALS === 'true'
  },

  jwt: {
    secret: process.env.JWT_SECRET || 'your-secret-key-change-in-production'
  }
};

export function validateConfig(): void {
  if (!config.jwt.secret || config.jwt.secret === 'your-secret-key-change-in-production') {
    console.warn('WARNING: Using default JWT secret. Set JWT_SECRET environment variable in production!');
  }

  console.log('Gateway Configuration:');
  console.log(`  Port: ${config.port}`);
  console.log(`  Auth Service: ${config.services.auth.url}`);
  console.log(`  Session Service: ${config.services.session.url}`);
  console.log(`  Sync Service: ${config.services.sync.url}`);
  console.log(`  Asset Service: ${config.services.asset.url}`);
  console.log(`  Presence Service: ${config.services.presence.url}`);
  console.log(`  Voice Service: ${config.services.voice.url}`);
  console.log(`  Analytics Service: ${config.services.analytics.url}`);
}
