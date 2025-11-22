import { Operation } from '../models/Operation';
import { OperationType, OTResult } from '../types';

/**
 * Operational Transform (OT) engine for conflict resolution
 * Implements a simplified OT algorithm for Unity object synchronization
 */
export class OperationalTransform {
  /**
   * Transform operation 'a' against operation 'b'
   * Returns transformed version of 'a' that can be applied after 'b'
   */
  transform(a: Operation, b: Operation): OTResult {
    // If operations don't affect same object, no transformation needed
    if (!a.affects(b)) {
      return {
        operation: a,
        transformed: false
      };
    }

    const transformed = a.clone();
    let hasConflict = false;

    // Handle different operation type combinations
    if (a.type === OperationType.DELETE && b.type === OperationType.DELETE) {
      // Both deleting same object - second delete is redundant
      return {
        operation: transformed,
        transformed: true,
        conflicts: [b]
      };
    }

    if (a.type === OperationType.DELETE && b.type === OperationType.UPDATE) {
      // Delete takes precedence over update
      return {
        operation: transformed,
        transformed: true
      };
    }

    if (a.type === OperationType.UPDATE && b.type === OperationType.DELETE) {
      // Object was deleted, update is invalid
      hasConflict = true;
      return {
        operation: transformed,
        transformed: true,
        conflicts: [b]
      };
    }

    if (a.type === OperationType.CREATE && b.type === OperationType.CREATE) {
      // Both creating same object - use timestamp to determine winner
      if (a.timestamp > b.timestamp) {
        transformed.type = OperationType.UPDATE; // Convert to update
      }
      return {
        operation: transformed,
        transformed: true,
        conflicts: hasConflict ? [b] : undefined
      };
    }

    if (a.type === OperationType.UPDATE && b.type === OperationType.UPDATE) {
      // Both updating - check if same path
      if (a.path === b.path) {
        // Last-write-wins based on timestamp
        if (a.timestamp < b.timestamp) {
          hasConflict = true;
        }
        return {
          operation: transformed,
          transformed: true,
          conflicts: hasConflict ? [b] : undefined
        };
      }

      // Different paths - both can apply
      return {
        operation: transformed,
        transformed: false
      };
    }

    if (a.type === OperationType.TRANSFORM && b.type === OperationType.TRANSFORM) {
      // Transform operations on same object
      transformed.version = Math.max(a.version, b.version) + 1;
      return {
        operation: transformed,
        transformed: true
      };
    }

    // Default: no transformation needed
    return {
      operation: transformed,
      transformed: false
    };
  }

  /**
   * Transform operation against a series of concurrent operations
   */
  transformAgainst(operation: Operation, concurrentOps: Operation[]): OTResult {
    let result: OTResult = {
      operation: operation,
      transformed: false
    };

    const allConflicts: Operation[] = [];

    for (const concurrentOp of concurrentOps) {
      result = this.transform(result.operation, concurrentOp);

      if (result.conflicts) {
        allConflicts.push(...result.conflicts);
      }
    }

    return {
      operation: result.operation,
      transformed: result.transformed,
      conflicts: allConflicts.length > 0 ? allConflicts : undefined
    };
  }

  /**
   * Check if operation is valid given current session state
   */
  isValid(operation: Operation, sessionVersion: number): boolean {
    // Operation version should be at or near session version
    const versionDiff = Math.abs(operation.version - sessionVersion);

    // Allow some version drift (up to 10 versions behind)
    return versionDiff <= 10;
  }

  /**
   * Compose two operations if possible
   * Combines operations that affect the same object/path
   */
  compose(a: Operation, b: Operation): Operation | null {
    // Can only compose operations on same object and path
    if (a.objectId !== b.objectId || a.path !== b.path) {
      return null;
    }

    // DELETE after any operation = DELETE
    if (b.type === OperationType.DELETE) {
      return b.clone();
    }

    // UPDATE after CREATE = CREATE with new value
    if (a.type === OperationType.CREATE && b.type === OperationType.UPDATE) {
      const composed = a.clone();
      composed.value = b.value;
      composed.version = b.version;
      composed.timestamp = b.timestamp;
      return composed;
    }

    // UPDATE after UPDATE = UPDATE with new value
    if (a.type === OperationType.UPDATE && b.type === OperationType.UPDATE) {
      const composed = a.clone();
      composed.value = b.value;
      composed.version = b.version;
      composed.timestamp = b.timestamp;
      return composed;
    }

    // CREATE after DELETE = CREATE
    if (a.type === OperationType.DELETE && b.type === OperationType.CREATE) {
      return b.clone();
    }

    return null;
  }

  /**
   * Compact operation history by composing operations
   */
  compactHistory(operations: Operation[]): Operation[] {
    if (operations.length === 0) {
      return [];
    }

    const compacted: Operation[] = [];
    let current = operations[0].clone();

    for (let i = 1; i < operations.length; i++) {
      const composed = this.compose(current, operations[i]);

      if (composed) {
        current = composed;
      } else {
        compacted.push(current);
        current = operations[i].clone();
      }
    }

    compacted.push(current);
    return compacted;
  }
}

// Singleton instance
let otEngine: OperationalTransform | null = null;

export function getOTEngine(): OperationalTransform {
  if (!otEngine) {
    otEngine = new OperationalTransform();
  }
  return otEngine;
}
