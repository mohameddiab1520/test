# Voice Chat Service

WebRTC-based voice chat service using Mediasoup SFU for Unity Collaboration Platform.

## Overview

The Voice Service provides high-quality, low-latency voice communication for collaboration sessions using Mediasoup as a Selective Forwarding Unit (SFU).

## Features

- **WebRTC Audio**: High-quality Opus codec audio
- **SFU Architecture**: Efficient media routing via Mediasoup
- **Low Latency**: Sub-100ms audio transmission
- **Scalable**: Support for 2-50 participants per channel
- **Audio Quality Monitoring**: Real-time quality metrics
- **Mute/Deafen Controls**: Individual audio controls
- **Socket.IO Integration**: Real-time signaling

## Technology Stack

- **Runtime**: Node.js 20+
- **Language**: TypeScript
- **Media Server**: Mediasoup (SFU)
- **Protocol**: WebRTC, Socket.IO
- **Codec**: Opus (48kHz, 2 channels)

## API Endpoints

```
POST   /api/v1/voice/channels                   - Create voice channel
GET    /api/v1/voice/channels                   - Get all channels
GET    /api/v1/voice/channels/:sessionId        - Get channel details
GET    /metrics                                  - Prometheus metrics
```

## Socket.IO Events

### Client → Server

```javascript
// Join voice channel
socket.emit('join-channel', {
  channel_id: 'voice_sess_abc123',
  user_id: 'user_abc',
  user_name: 'John Doe'
}, (response) => {
  console.log(response.rtpCapabilities);
});

// Create WebRTC transport
socket.emit('create-transport', {
  channel_id: 'voice_sess_abc123',
  user_id: 'user_abc'
}, (response) => {
  console.log(response.transport);
});

// Leave channel
socket.emit('leave-channel', {
  channel_id: 'voice_sess_abc123',
  user_id: 'user_abc'
});
```

### Server → Client

```javascript
// Participant joined
socket.on('participant-joined', (data) => {
  console.log(`${data.user_name} joined`);
});

// Participant left
socket.on('participant-left', (data) => {
  console.log(`User ${data.user_id} left`);
});
```

## Configuration

```bash
MEDIASOUP_WORKERS=4
RTC_MIN_PORT=10000
RTC_MAX_PORT=10100
MAX_PARTICIPANTS=50
```

## Development

```bash
npm install
npm run dev
```

## Build

```bash
npm run build
npm start
```

## Docker

```bash
docker build -t voice-service .
docker run -p 8086:8086 voice-service
```

## Metrics

- `voice_channels_created_total` - Total channels created
- `voice_channels_active` - Active voice channels
- `voice_participants_active` - Active participants
- `voice_audio_packets_lost_total` - Packets lost
- `voice_audio_latency_ms` - Audio latency
- `voice_audio_quality_score` - Quality score (0-100)

## License

MIT
