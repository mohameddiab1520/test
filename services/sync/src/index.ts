import { WebSocketServer, WebSocket } from 'ws';
import express from 'express';
import { createServer } from 'http';
import { v4 as uuidv4 } from 'uuid';

const app = express();
const server = createServer(app);
const wss = new WebSocketServer({ server });

const PORT = process.env.PORT || 8081;

// Store active connections
const connections = new Map<string, WebSocket>();

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'sync',
    version: '1.0.0',
    connections: connections.size
  });
});

// WebSocket connection handler
wss.on('connection', (ws: WebSocket, req) => {
  const clientId = uuidv4();
  connections.set(clientId, ws);

  console.log(`Client connected: ${clientId}. Total connections: ${connections.size}`);

  // Send welcome message
  ws.send(JSON.stringify({
    type: 'connected',
    clientId,
    message: 'Connected to Sync Service'
  }));

  // Handle incoming messages
  ws.on('message', (data: Buffer) => {
    try {
      const message = JSON.parse(data.toString());
      console.log('Received message:', message);

      // Echo message back to sender
      ws.send(JSON.stringify({
        type: 'ack',
        success: true,
        originalMessage: message
      }));

      // Broadcast to other clients
      connections.forEach((client, id) => {
        if (id !== clientId && client.readyState === WebSocket.OPEN) {
          client.send(JSON.stringify({
            type: 'sync',
            from: clientId,
            data: message
          }));
        }
      });

    } catch (error) {
      console.error('Error processing message:', error);
      ws.send(JSON.stringify({
        type: 'error',
        message: 'Invalid message format'
      }));
    }
  });

  // Handle ping/pong
  ws.on('ping', () => {
    ws.pong();
  });

  // Handle disconnection
  ws.on('close', () => {
    connections.delete(clientId);
    console.log(`Client disconnected: ${clientId}. Total connections: ${connections.size}`);
  });

  // Handle errors
  ws.on('error', (error) => {
    console.error(`WebSocket error for client ${clientId}:`, error);
  });
});

// Start server
server.listen(PORT, () => {
  console.log(`Sync Service listening on port ${PORT}`);
  console.log(`WebSocket endpoint: ws://localhost:${PORT}`);
  console.log(`Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM signal received: closing HTTP server');
  server.close(() => {
    console.log('HTTP server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('SIGINT signal received: closing HTTP server');
  server.close(() => {
    console.log('HTTP server closed');
    process.exit(0);
  });
});
