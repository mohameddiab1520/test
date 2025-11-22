import { SyncOperation, OperationType, OperationData } from '../types';
import { v4 as uuidv4 } from 'uuid';

export class Operation {
  id: string;
  sessionId: string;
  userId: string;
  type: OperationType;
  objectId: string;
  path?: string;
  value?: any;
  version: number;
  timestamp: Date;
  metadata: Record<string, any>;

  constructor(
    sessionId: string,
    userId: string,
    data: OperationData
  ) {
    this.id = uuidv4();
    this.sessionId = sessionId;
    this.userId = userId;
    this.type = data.type;
    this.objectId = data.objectId;
    this.path = data.path;
    this.value = data.value;
    this.version = data.version || 0;
    this.timestamp = new Date();
    this.metadata = data.metadata || {};
  }

  toJSON(): SyncOperation {
    return {
      id: this.id,
      sessionId: this.sessionId,
      userId: this.userId,
      type: this.type,
      objectId: this.objectId,
      path: this.path,
      value: this.value,
      version: this.version,
      timestamp: this.timestamp,
      metadata: this.metadata
    };
  }

  static fromJSON(data: SyncOperation): Operation {
    const op = Object.create(Operation.prototype);
    Object.assign(op, data);
    op.timestamp = new Date(data.timestamp);
    return op;
  }

  clone(): Operation {
    const op = new Operation(this.sessionId, this.userId, {
      type: this.type,
      objectId: this.objectId,
      path: this.path,
      value: this.value,
      version: this.version,
      metadata: { ...this.metadata }
    });
    op.id = this.id;
    op.timestamp = new Date(this.timestamp);
    return op;
  }

  // Check if this operation affects the same object/path as another
  affects(other: Operation): boolean {
    if (this.objectId !== other.objectId) {
      return false;
    }

    // Same object, check path
    if (!this.path || !other.path) {
      return true; // If no path specified, affects entire object
    }

    return this.path === other.path ||
           this.path.startsWith(other.path + '.') ||
           other.path.startsWith(this.path + '.');
  }

  // Check if operations are on same object and path
  conflictsWith(other: Operation): boolean {
    return this.objectId === other.objectId &&
           this.path === other.path &&
           this.type !== OperationType.DELETE &&
           other.type !== OperationType.DELETE;
  }
}
