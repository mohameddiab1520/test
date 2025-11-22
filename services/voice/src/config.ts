export const config = {
  serviceName: process.env.SERVICE_NAME || 'voice-service',
  servicePort: parseInt(process.env.SERVICE_PORT || '8086'),
  environment: process.env.ENVIRONMENT || 'development',

  // PostgreSQL
  postgres: {
    host: process.env.POSTGRES_HOST || 'localhost',
    port: parseInt(process.env.POSTGRES_PORT || '5432'),
    database: process.env.POSTGRES_DB || 'collab_dev',
    user: process.env.POSTGRES_USER || 'dev',
    password: process.env.POSTGRES_PASSWORD || 'devpass',
  },

  // Redis
  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: parseInt(process.env.REDIS_PORT || '6379'),
    password: process.env.REDIS_PASSWORD || '',
  },

  // Mediasoup
  mediasoup: {
    numWorkers: parseInt(process.env.MEDIASOUP_WORKERS || '4'),
    workerSettings: {
      rtcMinPort: parseInt(process.env.RTC_MIN_PORT || '10000'),
      rtcMaxPort: parseInt(process.env.RTC_MAX_PORT || '10100'),
      logLevel: process.env.MEDIASOUP_LOG_LEVEL || 'warn',
      logTags: ['info', 'ice', 'dtls', 'rtp', 'srtp', 'rtcp'],
    },
    routerOptions: {
      mediaCodecs: [
        {
          kind: 'audio',
          mimeType: 'audio/opus',
          clockRate: 48000,
          channels: 2,
        },
      ],
    },
  },

  // Voice settings
  maxParticipantsPerChannel: parseInt(process.env.MAX_PARTICIPANTS || '50'),
  audioLevelInterval: parseInt(process.env.AUDIO_LEVEL_INTERVAL || '1000'),
};
