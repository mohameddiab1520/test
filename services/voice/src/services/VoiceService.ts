import * as mediasoup from 'mediasoup';
import { types as mediasoupTypes } from 'mediasoup';
import { config } from '../config';
import { MetricsService } from './MetricsService';
import { logger } from '../utils/logger';

export interface VoiceChannel {
  channelId: string;
  sessionId: string;
  router: mediasoupTypes.Router;
  participants: Map<string, VoiceParticipant>;
  recordingEnabled: boolean;
  createdAt: Date;
}

export interface VoiceParticipant {
  userId: string;
  userName: string;
  transport?: mediasoupTypes.WebRtcTransport;
  producer?: mediasoupTypes.Producer;
  consumers: Map<string, mediasoupTypes.Consumer>;
  muted: boolean;
  deafened: boolean;
  audioLevel: number;
  joinedAt: Date;
}

export class VoiceService {
  private workers: mediasoupTypes.Worker[] = [];
  private nextWorkerIndex = 0;
  private channels: Map<string, VoiceChannel> = new Map();
  private metrics: MetricsService;

  constructor(metrics: MetricsService) {
    this.metrics = metrics;
  }

  async initialize() {
    logger.info('Initializing Mediasoup workers...');

    for (let i = 0; i < config.mediasoup.numWorkers; i++) {
      const worker = await mediasoup.createWorker({
        rtcMinPort: config.mediasoup.workerSettings.rtcMinPort,
        rtcMaxPort: config.mediasoup.workerSettings.rtcMaxPort,
        logLevel: config.mediasoup.workerSettings.logLevel as any,
        logTags: config.mediasoup.workerSettings.logTags as any[],
      });

      worker.on('died', () => {
        logger.error(`Mediasoup worker died [pid:${worker.pid}]`);
        process.exit(1);
      });

      this.workers.push(worker);
      logger.info(`Mediasoup worker created [pid:${worker.pid}]`);
    }

    logger.info(`Initialized ${this.workers.length} Mediasoup workers`);
  }

  async createChannel(sessionId: string): Promise<VoiceChannel> {
    const channelId = `voice_${sessionId}`;

    if (this.channels.has(channelId)) {
      return this.channels.get(channelId)!;
    }

    const worker = this.getNextWorker();
    const router = await worker.createRouter({
      mediaCodecs: config.mediasoup.routerOptions.mediaCodecs as any[],
    });

    const channel: VoiceChannel = {
      channelId,
      sessionId,
      router,
      participants: new Map(),
      recordingEnabled: false,
      createdAt: new Date(),
    };

    this.channels.set(channelId, channel);
    this.metrics.incrementChannelsCreated();

    logger.info(`Voice channel created: ${channelId}`);

    return channel;
  }

  async joinChannel(
    channelId: string,
    userId: string,
    userName: string
  ): Promise<{ rtpCapabilities: mediasoupTypes.RtpCapabilities }> {
    const channel = this.channels.get(channelId);
    if (!channel) {
      throw new Error('Channel not found');
    }

    if (channel.participants.size >= config.maxParticipantsPerChannel) {
      throw new Error('Channel is full');
    }

    const participant: VoiceParticipant = {
      userId,
      userName,
      consumers: new Map(),
      muted: false,
      deafened: false,
      audioLevel: 0,
      joinedAt: new Date(),
    };

    channel.participants.set(userId, participant);
    this.metrics.updateActiveParticipants(this.getTotalParticipants());

    logger.info(`User ${userId} joined channel ${channelId}`);

    return {
      rtpCapabilities: channel.router.rtpCapabilities,
    };
  }

  async createTransport(channelId: string, userId: string): Promise<any> {
    const channel = this.channels.get(channelId);
    if (!channel) {
      throw new Error('Channel not found');
    }

    const participant = channel.participants.get(userId);
    if (!participant) {
      throw new Error('Participant not found');
    }

    const transport = await channel.router.createWebRtcTransport({
      listenIps: [{ ip: '0.0.0.0', announcedIp: process.env.ANNOUNCED_IP }],
      enableUdp: true,
      enableTcp: true,
      preferUdp: true,
    });

    participant.transport = transport;

    return {
      id: transport.id,
      iceParameters: transport.iceParameters,
      iceCandidates: transport.iceCandidates,
      dtlsParameters: transport.dtlsParameters,
    };
  }

  async leaveChannel(channelId: string, userId: string) {
    const channel = this.channels.get(channelId);
    if (!channel) return;

    const participant = channel.participants.get(userId);
    if (participant) {
      // Close transport
      participant.transport?.close();

      // Close producer
      participant.producer?.close();

      // Close all consumers
      participant.consumers.forEach((consumer) => consumer.close());

      channel.participants.delete(userId);
      this.metrics.updateActiveParticipants(this.getTotalParticipants());
    }

    // Close channel if empty
    if (channel.participants.size === 0) {
      channel.router.close();
      this.channels.delete(channelId);
      logger.info(`Voice channel closed: ${channelId}`);
    }
  }

  getChannel(channelId: string): VoiceChannel | undefined {
    return this.channels.get(channelId);
  }

  getChannels(): VoiceChannel[] {
    return Array.from(this.channels.values());
  }

  private getNextWorker(): mediasoupTypes.Worker {
    const worker = this.workers[this.nextWorkerIndex];
    this.nextWorkerIndex = (this.nextWorkerIndex + 1) % this.workers.length;
    return worker;
  }

  private getTotalParticipants(): number {
    let total = 0;
    this.channels.forEach((channel) => {
      total += channel.participants.size;
    });
    return total;
  }

  async shutdown() {
    logger.info('Shutting down Voice Service...');

    // Close all channels
    this.channels.forEach((channel) => {
      channel.router.close();
    });

    // Close all workers
    for (const worker of this.workers) {
      worker.close();
    }

    logger.info('Voice Service shut down successfully');
  }
}
