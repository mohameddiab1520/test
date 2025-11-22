import { Server as SocketIOServer, Socket } from 'socket.io';
import { VoiceService } from '../services/VoiceService';
import { logger } from '../utils/logger';

export function setupSocketHandlers(io: SocketIOServer, voiceService: VoiceService) {
  io.on('connection', (socket: Socket) => {
    logger.info(`Socket connected: ${socket.id}`);

    socket.on('join-channel', async (data, callback) => {
      try {
        const { channel_id, user_id, user_name } = data;

        const result = await voiceService.joinChannel(channel_id, user_id, user_name);

        socket.join(channel_id);

        callback({ success: true, rtpCapabilities: result.rtpCapabilities });

        // Notify others
        socket.to(channel_id).emit('participant-joined', { user_id, user_name });
      } catch (error: any) {
        logger.error('Error joining channel', { error });
        callback({ success: false, error: error.message });
      }
    });

    socket.on('create-transport', async (data, callback) => {
      try {
        const { channel_id, user_id } = data;

        const transport = await voiceService.createTransport(channel_id, user_id);

        callback({ success: true, transport });
      } catch (error: any) {
        logger.error('Error creating transport', { error });
        callback({ success: false, error: error.message });
      }
    });

    socket.on('leave-channel', async (data) => {
      const { channel_id, user_id } = data;

      await voiceService.leaveChannel(channel_id, user_id);

      socket.leave(channel_id);

      // Notify others
      socket.to(channel_id).emit('participant-left', { user_id });
    });

    socket.on('disconnect', () => {
      logger.info(`Socket disconnected: ${socket.id}`);
    });
  });
}
