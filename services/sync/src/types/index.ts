// Type definitions for Sync Service

export interface User {
  id: string;
  email: string;
  displayName?: string;
}

export interface Session {
  id: string;
  projectId: string;
  name: string;
  ownerId: string;
  participantIds: string[];
  createdAt: Date;
  updatedAt: Date;
}

export interface SyncOperation {
  id: string;
  sessionId: string;
  userId: string;
  type: OperationType;
  objectId: string;
  path?: string;
  value?: any;
  version: number;
  timestamp: Date;
  metadata?: Record<string, any>;
}

export enum OperationType {
  CREATE = 'create',
  UPDATE = 'update',
  DELETE = 'delete',
  TRANSFORM = 'transform'
}

export interface ClientMessage {
  type: MessageType;
  sessionId?: string;
  token?: string;
  operation?: OperationData;
  data?: any;
}

export interface ServerMessage {
  type: MessageType;
  clientId?: string;
  operation?: SyncOperation;
  error?: string;
  message?: string;
  data?: any;
}

export enum MessageType {
  // Client messages
  JOIN = 'join',
  LEAVE = 'leave',
  OPERATION = 'operation',
  PING = 'ping',

  // Server messages
  CONNECTED = 'connected',
  JOINED = 'joined',
  LEFT = 'left',
  SYNC = 'sync',
  ACK = 'ack',
  ERROR = 'error',
  PONG = 'pong'
}

export interface OperationData {
  type: OperationType;
  objectId: string;
  path?: string;
  value?: any;
  version?: number;
  metadata?: Record<string, any>;
}

export interface ClientConnection {
  id: string;
  userId?: string;
  sessionId?: string;
  ws: any; // WebSocket
  connectedAt: Date;
  lastActivity: Date;
}

export interface SessionState {
  sessionId: string;
  clients: Set<string>;
  operations: SyncOperation[];
  version: number;
  lastUpdated: Date;
}

export interface OTResult {
  operation: SyncOperation;
  transformed: boolean;
  conflicts?: SyncOperation[];
}

export interface RedisConfig {
  host: string;
  port: number;
  password?: string;
  db?: number;
}

export interface SyncConfig {
  port: number;
  redis: RedisConfig;
  jwtSecret: string;
  logLevel: string;
  maxOperationsHistory: number;
  operationTimeout: number;
  heartbeatInterval: number;
}
