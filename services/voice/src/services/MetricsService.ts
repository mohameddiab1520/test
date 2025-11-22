import { Counter, Gauge, Histogram, Registry } from 'prom-client';

export class MetricsService {
  private registry: Registry;
  private channelsCreated: Counter;
  private activeChannels: Gauge;
  private activeParticipants: Gauge;
  private audioPacketsLost: Counter;
  private audioLatency: Histogram;
  private audioQuality: Gauge;

  constructor() {
    this.registry = new Registry();

    this.channelsCreated = new Counter({
      name: 'voice_channels_created_total',
      help: 'Total number of voice channels created',
      registers: [this.registry],
    });

    this.activeChannels = new Gauge({
      name: 'voice_channels_active',
      help: 'Number of active voice channels',
      registers: [this.registry],
    });

    this.activeParticipants = new Gauge({
      name: 'voice_participants_active',
      help: 'Number of active voice participants',
      registers: [this.registry],
    });

    this.audioPacketsLost = new Counter({
      name: 'voice_audio_packets_lost_total',
      help: 'Total number of audio packets lost',
      labelNames: ['user_id'],
      registers: [this.registry],
    });

    this.audioLatency = new Histogram({
      name: 'voice_audio_latency_ms',
      help: 'Audio latency in milliseconds',
      buckets: [10, 25, 50, 100, 250, 500, 1000],
      registers: [this.registry],
    });

    this.audioQuality = new Gauge({
      name: 'voice_audio_quality_score',
      help: 'Audio quality score (0-100)',
      labelNames: ['user_id'],
      registers: [this.registry],
    });
  }

  incrementChannelsCreated() {
    this.channelsCreated.inc();
  }

  updateActiveChannels(count: number) {
    this.activeChannels.set(count);
  }

  updateActiveParticipants(count: number) {
    this.activeParticipants.set(count);
  }

  incrementPacketsLost(userId: string) {
    this.audioPacketsLost.inc({ user_id: userId });
  }

  recordLatency(latency: number) {
    this.audioLatency.observe(latency);
  }

  updateQuality(userId: string, quality: number) {
    this.audioQuality.set({ user_id: userId }, quality);
  }

  async getMetrics(): Promise<string> {
    return this.registry.metrics();
  }
}
