import { Express, Request, Response } from 'express';
import { VoiceService } from '../services/VoiceService';
import { MetricsService } from '../services/MetricsService';

export function setupRoutes(
  app: Express,
  voiceService: VoiceService,
  metricsService: MetricsService
) {
  // Get all channels
  app.get('/api/v1/voice/channels', (req: Request, res: Response) => {
    const channels = voiceService.getChannels().map((channel) => ({
      channel_id: channel.channelId,
      session_id: channel.sessionId,
      participants: channel.participants.size,
      recording_enabled: channel.recordingEnabled,
      created_at: channel.createdAt,
    }));

    res.json(channels);
  });

  // Get channel details
  app.get('/api/v1/voice/channels/:sessionId', (req: Request, res: Response) => {
    const channelId = `voice_${req.params.sessionId}`;
    const channel = voiceService.getChannel(channelId);

    if (!channel) {
      return res.status(404).json({ error: 'Channel not found' });
    }

    const participants = Array.from(channel.participants.values()).map((p) => ({
      user_id: p.userId,
      user_name: p.userName,
      muted: p.muted,
      deafened: p.deafened,
      audio_level: p.audioLevel,
      joined_at: p.joinedAt,
    }));

    res.json({
      channel_id: channel.channelId,
      session_id: channel.sessionId,
      participants,
      recording_enabled: channel.recordingEnabled,
      created_at: channel.createdAt,
    });
  });

  // Create channel
  app.post('/api/v1/voice/channels', async (req: Request, res: Response) => {
    try {
      const { session_id } = req.body;

      if (!session_id) {
        return res.status(400).json({ error: 'session_id is required' });
      }

      const channel = await voiceService.createChannel(session_id);

      res.json({
        channel_id: channel.channelId,
        session_id: channel.sessionId,
        created_at: channel.createdAt,
      });
    } catch (error: any) {
      res.status(500).json({ error: error.message });
    }
  });

  // Prometheus metrics
  app.get('/metrics', async (req: Request, res: Response) => {
    res.set('Content-Type', 'text/plain');
    res.send(await metricsService.getMetrics());
  });
}
