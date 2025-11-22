import { WebSocket } from 'ws';
import { v4 as uuidv4 } from 'uuid';
import {
  ClientConnection,
  SessionState,
  ClientMessage,
  ServerMessage,
  MessageType,
  OperationData
} from '../types';
import { Operation } from '../models/Operation';
import { getRedisClient } from '../redis/RedisClient';
import { getOTEngine } from '../ot/OperationalTransform';
import { config } from '../config';

export class SessionManager {
  private connections: Map<string, ClientConnection> = new Map();
  private sessions: Map<string, SessionState> = new Map();
  private redis = getRedisClient();
  private otEngine = getOTEngine();

  /**
   * Handle new WebSocket connection
   */
  async handleConnection(ws: WebSocket): Promise<string> {
    const clientId = uuidv4();
    const connection: ClientConnection = {
      id: clientId,
      ws,
      connectedAt: new Date(),
      lastActivity: new Date()
    };

    this.connections.set(clientId, connection);

    // Send welcome message
    this.sendToClient(clientId, {
      type: MessageType.CONNECTED,
      clientId,
      message: 'Connected to Sync Service'
    });

    // Setup message handler
    ws.on('message', async (data: Buffer) => {
      await this.handleMessage(clientId, data);
    });

    // Setup close handler
    ws.on('close', async () => {
      await this.handleDisconnection(clientId);
    });

    // Setup error handler
    ws.on('error', (error) => {
      console.error(`WebSocket error for client ${clientId}:`, error);
    });

    console.log(`Client connected: ${clientId}`);
    return clientId;
  }

  /**
   * Handle incoming message from client
   */
  private async handleMessage(clientId: string, data: Buffer): Promise<void> {
    try {
      const connection = this.connections.get(clientId);
      if (!connection) {
        return;
      }

      connection.lastActivity = new Date();

      const message: ClientMessage = JSON.parse(data.toString());

      switch (message.type) {
        case MessageType.JOIN:
          await this.handleJoin(clientId, message);
          break;

        case MessageType.LEAVE:
          await this.handleLeave(clientId, message);
          break;

        case MessageType.OPERATION:
          await this.handleOperation(clientId, message);
          break;

        case MessageType.PING:
          this.handlePing(clientId);
          break;

        default:
          this.sendError(clientId, `Unknown message type: ${message.type}`);
      }
    } catch (error) {
      console.error(`Error handling message from ${clientId}:`, error);
      this.sendError(clientId, 'Invalid message format');
    }
  }

  /**
   * Handle client joining a session
   */
  private async handleJoin(clientId: string, message: ClientMessage): Promise<void> {
    const { sessionId, token } = message;

    if (!sessionId) {
      this.sendError(clientId, 'Session ID required');
      return;
    }

    // TODO: Validate token and extract user info
    // For now, we'll skip auth validation

    const connection = this.connections.get(clientId);
    if (!connection) {
      return;
    }

    // Update connection with session info
    connection.sessionId = sessionId;
    connection.userId = 'user-' + clientId.substring(0, 8); // Mock user ID

    // Get or create session state
    let session = this.sessions.get(sessionId);

    if (!session) {
      // Try to load from Redis
      session = await this.redis.getSessionState(sessionId);

      if (!session) {
        // Create new session
        session = {
          sessionId,
          clients: new Set(),
          operations: [],
          version: 0,
          lastUpdated: new Date()
        };
      }

      this.sessions.set(sessionId, session);
    }

    // Add client to session
    session.clients.add(clientId);
    await this.redis.addClientToSession(sessionId, clientId);

    // Send join confirmation
    this.sendToClient(clientId, {
      type: MessageType.JOINED,
      data: {
        sessionId,
        version: session.version,
        clientCount: session.clients.size
      }
    });

    // Subscribe to Redis pub/sub for this session
    await this.redis.subscribeToSession(sessionId, (operation) => {
      this.broadcastOperation(sessionId, operation);
    });

    console.log(`Client ${clientId} joined session ${sessionId}`);
  }

  /**
   * Handle client leaving a session
   */
  private async handleLeave(clientId: string, message: ClientMessage): Promise<void> {
    const connection = this.connections.get(clientId);
    if (!connection || !connection.sessionId) {
      return;
    }

    await this.removeClientFromSession(clientId, connection.sessionId);

    this.sendToClient(clientId, {
      type: MessageType.LEFT,
      message: 'Left session successfully'
    });
  }

  /**
   * Handle operation from client
   */
  private async handleOperation(clientId: string, message: ClientMessage): Promise<void> {
    const connection = this.connections.get(clientId);

    if (!connection || !connection.sessionId || !connection.userId) {
      this.sendError(clientId, 'Must join a session first');
      return;
    }

    if (!message.operation) {
      this.sendError(clientId, 'Operation data required');
      return;
    }

    const session = this.sessions.get(connection.sessionId);
    if (!session) {
      this.sendError(clientId, 'Session not found');
      return;
    }

    try {
      // Create operation
      const operation = new Operation(
        connection.sessionId,
        connection.userId,
        message.operation
      );

      // Validate operation
      if (!this.otEngine.isValid(operation, session.version)) {
        this.sendError(clientId, 'Operation version mismatch');
        return;
      }

      // Get concurrent operations
      const concurrentOps = session.operations.filter(
        op => op.version >= operation.version
      );

      // Transform against concurrent operations
      const result = this.otEngine.transformAgainst(operation, concurrentOps);

      // Update version
      result.operation.version = session.version + 1;

      // Add to session history
      session.operations.push(result.operation.toJSON());
      session.version++;
      session.lastUpdated = new Date();

      // Keep only recent operations in memory
      if (session.operations.length > config.maxOperationsHistory) {
        session.operations = session.operations.slice(-config.maxOperationsHistory);
      }

      // Save to Redis
      await this.redis.addOperation(connection.sessionId, result.operation.toJSON());
      await this.redis.setSessionState(session);

      // Publish to Redis for other instances
      await this.redis.publishOperation(connection.sessionId, result.operation.toJSON());

      // Send acknowledgment
      this.sendToClient(clientId, {
        type: MessageType.ACK,
        data: {
          operationId: result.operation.id,
          version: result.operation.version,
          transformed: result.transformed,
          conflicts: result.conflicts?.length || 0
        }
      });

      // Broadcast to other clients in session
      this.broadcastToSession(
        connection.sessionId,
        {
          type: MessageType.SYNC,
          operation: result.operation.toJSON()
        },
        clientId // Exclude sender
      );

      console.log(`Operation applied: ${result.operation.id} in session ${connection.sessionId}`);
    } catch (error) {
      console.error('Error processing operation:', error);
      this.sendError(clientId, 'Failed to process operation');
    }
  }

  /**
   * Handle ping from client
   */
  private handlePing(clientId: string): void {
    const connection = this.connections.get(clientId);
    if (!connection) {
      return;
    }

    // Update heartbeat
    if (connection.sessionId) {
      this.redis.updateClientHeartbeat(clientId, connection.sessionId);
    }

    this.sendToClient(clientId, {
      type: MessageType.PONG
    });
  }

  /**
   * Handle client disconnection
   */
  private async handleDisconnection(clientId: string): Promise<void> {
    const connection = this.connections.get(clientId);

    if (connection && connection.sessionId) {
      await this.removeClientFromSession(clientId, connection.sessionId);
    }

    this.connections.delete(clientId);
    console.log(`Client disconnected: ${clientId}`);
  }

  /**
   * Remove client from session
   */
  private async removeClientFromSession(clientId: string, sessionId: string): Promise<void> {
    const session = this.sessions.get(sessionId);

    if (session) {
      session.clients.delete(clientId);

      // If no more clients, cleanup session
      if (session.clients.size === 0) {
        await this.redis.unsubscribeFromSession(sessionId);
        this.sessions.delete(sessionId);
      }
    }

    await this.redis.removeClientFromSession(sessionId, clientId);
  }

  /**
   * Send message to specific client
   */
  private sendToClient(clientId: string, message: ServerMessage): void {
    const connection = this.connections.get(clientId);

    if (connection && connection.ws.readyState === WebSocket.OPEN) {
      connection.ws.send(JSON.stringify(message));
    }
  }

  /**
   * Send error to client
   */
  private sendError(clientId: string, error: string): void {
    this.sendToClient(clientId, {
      type: MessageType.ERROR,
      error
    });
  }

  /**
   * Broadcast message to all clients in session
   */
  private broadcastToSession(
    sessionId: string,
    message: ServerMessage,
    excludeClientId?: string
  ): void {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return;
    }

    session.clients.forEach(clientId => {
      if (clientId !== excludeClientId) {
        this.sendToClient(clientId, message);
      }
    });
  }

  /**
   * Broadcast operation to session (from Redis pub/sub)
   */
  private broadcastOperation(sessionId: string, operation: any): void {
    this.broadcastToSession(sessionId, {
      type: MessageType.SYNC,
      operation
    });
  }

  /**
   * Get connection count
   */
  getConnectionCount(): number {
    return this.connections.size;
  }

  /**
   * Get session count
   */
  getSessionCount(): number {
    return this.sessions.size;
  }

  /**
   * Cleanup and shutdown
   */
  async shutdown(): Promise<void> {
    console.log('Shutting down SessionManager...');

    // Close all connections
    this.connections.forEach((connection, clientId) => {
      if (connection.ws.readyState === WebSocket.OPEN) {
        connection.ws.close();
      }
    });

    this.connections.clear();
    this.sessions.clear();

    console.log('SessionManager shutdown complete');
  }
}
