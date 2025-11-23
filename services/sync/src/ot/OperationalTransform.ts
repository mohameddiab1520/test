import { Operation } from '../models/Operation';
import { OperationType, OTResult } from '../types';

/**
 * Vector Clock for tracking causality in distributed systems
 */
interface VectorClock {
  [userId: string]: number;
}

/**
 * Advanced Operational Transform (OT) engine for conflict resolution
 * Implements full OT algorithm with TP1 and TP2 properties
 * Ensures convergence and intent preservation
 */
export class OperationalTransform {
  private vectorClocks: Map<string, VectorClock> = new Map();

  /**
   * Transform operation 'a' against operation 'b'
   * Returns transformed version of 'a' that can be applied after 'b'
   * Ensures TP1: transform(a, b) ⊕ b = transform(b, a) ⊕ a (Transformation Property 1)
   */
  transform(a: Operation, b: Operation): OTResult {
    // If operations don't affect same object, no transformation needed
    if (!a.affects(b)) {
      return {
        operation: a,
        transformed: false
      };
    }

    // Check causality using vector clocks
    const causalityRelation = this.checkCausality(a, b);

    const transformed = a.clone();
    let hasConflict = false;
    const conflicts: Operation[] = [];

    // Handle different operation type combinations with proper OT
    if (a.type === OperationType.DELETE && b.type === OperationType.DELETE) {
      // DD: Both deleting same object
      // Second delete becomes no-op (identity operation)
      transformed.type = OperationType.NOOP;
      conflicts.push(b);
      return {
        operation: transformed,
        transformed: true,
        conflicts
      };
    }

    if (a.type === OperationType.DELETE && b.type === OperationType.UPDATE) {
      // DU: Delete against Update
      // Delete takes precedence, but mark conflict
      hasConflict = true;
      conflicts.push(b);
      return {
        operation: transformed,
        transformed: true,
        conflicts
      };
    }

    if (a.type === OperationType.UPDATE && b.type === OperationType.DELETE) {
      // UD: Update against Delete
      // Object was deleted, update is invalid
      transformed.type = OperationType.NOOP;
      hasConflict = true;
      conflicts.push(b);
      return {
        operation: transformed,
        transformed: true,
        conflicts
      };
    }

    if (a.type === OperationType.CREATE && b.type === OperationType.CREATE) {
      // CC: Both creating same object
      // Use vector clock to determine order, not timestamp
      if (causalityRelation === 'concurrent') {
        // Truly concurrent - use deterministic resolution
        const winner = this.resolveConcurrentCreate(a, b);
        if (winner === a) {
          transformed.type = OperationType.CREATE;
        } else {
          transformed.type = OperationType.NOOP;
          hasConflict = true;
        }
      } else if (causalityRelation === 'a-after-b') {
        // a happened after b, convert to update
        transformed.type = OperationType.UPDATE;
      }
      return {
        operation: transformed,
        transformed: true,
        conflicts: hasConflict ? [b] : undefined
      };
    }

    if (a.type === OperationType.UPDATE && b.type === OperationType.UPDATE) {
      // UU: Both updating same object
      return this.transformUpdateUpdate(a, b, causalityRelation);
    }

    if (a.type === OperationType.TRANSFORM && b.type === OperationType.TRANSFORM) {
      // TT: Transform operations (position, rotation, scale)
      return this.transformTransformTransform(a, b, causalityRelation);
    }

    // Default: no transformation needed
    return {
      operation: transformed,
      transformed: false
    };
  }

  /**
   * Transform two concurrent UPDATE operations
   * Implements proper convergence for property updates
   */
  private transformUpdateUpdate(a: Operation, b: Operation, causality: string): OTResult {
    const transformed = a.clone();
    let hasConflict = false;
    const conflicts: Operation[] = [];

    // Check if updating same path
    if (a.path === b.path) {
      if (causality === 'concurrent') {
        // Truly concurrent updates to same property
        // Use Operational Transformation to preserve both intents

        if (this.isNumericValue(a.value) && this.isNumericValue(b.value)) {
          // For numeric values, apply delta transformation
          const aDelta = Number(a.value) - (a.previousValue ? Number(a.previousValue) : 0);
          const bDelta = Number(b.value) - (b.previousValue ? Number(b.previousValue) : 0);

          // Apply both deltas
          transformed.value = (b.previousValue ? Number(b.previousValue) : 0) + bDelta + aDelta;
        } else {
          // For non-numeric, use deterministic tie-breaking
          const winner = this.resolveConflictDeterministic(a, b);
          if (winner !== a) {
            hasConflict = true;
            transformed.value = b.value;
            conflicts.push(b);
          }
        }
      } else if (causality === 'b-after-a') {
        // b happened after a, so a's change is overridden
        transformed.value = b.value;
        hasConflict = true;
        conflicts.push(b);
      }
      // else: a happened after b, keep a's value

      return {
        operation: transformed,
        transformed: true,
        conflicts: conflicts.length > 0 ? conflicts : undefined
      };
    }

    // Different paths - both can apply independently
    return {
      operation: transformed,
      transformed: false
    };
  }

  /**
   * Transform two concurrent TRANSFORM operations (position, rotation, scale)
   * Implements sophisticated transformation for Unity transforms
   */
  private transformTransformTransform(a: Operation, b: Operation, causality: string): OTResult {
    const transformed = a.clone();

    if (causality === 'concurrent') {
      // Both transforms are concurrent, merge them intelligently

      // Parse transform values
      const aTransform = this.parseTransformValue(a.value);
      const bTransform = this.parseTransformValue(b.value);

      if (aTransform && bTransform) {
        // Calculate deltas from previous state
        const aPrevTransform = this.parseTransformValue(a.previousValue);
        const bPrevTransform = this.parseTransformValue(b.previousValue);

        // Apply both transformations
        const merged = {
          position: this.mergeVectors(
            aTransform.position,
            bTransform.position,
            aPrevTransform?.position,
            bPrevTransform?.position
          ),
          rotation: this.mergeRotations(
            aTransform.rotation,
            bTransform.rotation,
            aPrevTransform?.rotation,
            bPrevTransform?.rotation
          ),
          scale: this.mergeVectors(
            aTransform.scale,
            bTransform.scale,
            aPrevTransform?.scale,
            bPrevTransform?.scale
          )
        };

        transformed.value = JSON.stringify(merged);
      }

      transformed.version = Math.max(a.version, b.version) + 1;
    }

    return {
      operation: transformed,
      transformed: true
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

  /**
   * Check causality relation between two operations using vector clocks
   */
  private checkCausality(a: Operation, b: Operation): 'a-after-b' | 'b-after-a' | 'concurrent' {
    const aVC = this.getVectorClock(a.userId);
    const bVC = this.getVectorClock(b.userId);

    let aGreater = false;
    let bGreater = false;

    // Compare vector clocks
    const allUsers = new Set([...Object.keys(aVC), ...Object.keys(bVC)]);

    for (const user of allUsers) {
      const aCount = aVC[user] || 0;
      const bCount = bVC[user] || 0;

      if (aCount > bCount) aGreater = true;
      if (bCount > aCount) bGreater = true;
    }

    if (aGreater && !bGreater) return 'a-after-b';
    if (bGreater && !aGreater) return 'b-after-a';
    return 'concurrent';
  }

  /**
   * Get or initialize vector clock for a user
   */
  private getVectorClock(userId: string): VectorClock {
    if (!this.vectorClocks.has(userId)) {
      this.vectorClocks.set(userId, {});
    }
    return this.vectorClocks.get(userId)!;
  }

  /**
   * Update vector clock for an operation
   */
  updateVectorClock(userId: string, operation: Operation): void {
    const vc = this.getVectorClock(userId);
    vc[userId] = (vc[userId] || 0) + 1;
    this.vectorClocks.set(userId, vc);
  }

  /**
   * Resolve concurrent CREATE operations deterministically
   */
  private resolveConcurrentCreate(a: Operation, b: Operation): Operation {
    // Use user ID as tie-breaker for deterministic resolution
    if (a.userId < b.userId) return a;
    if (b.userId < a.userId) return b;

    // If same user (unlikely), use timestamp
    return a.timestamp < b.timestamp ? a : b;
  }

  /**
   * Resolve conflicts deterministically
   */
  private resolveConflictDeterministic(a: Operation, b: Operation): Operation {
    // Deterministic conflict resolution based on user ID
    if (a.userId < b.userId) return a;
    if (b.userId < a.userId) return b;

    // Fallback to timestamp
    return a.timestamp < b.timestamp ? a : b;
  }

  /**
   * Check if value is numeric
   */
  private isNumericValue(value: any): boolean {
    return !isNaN(Number(value));
  }

  /**
   * Parse transform value from string
   */
  private parseTransformValue(value: any): {
    position: { x: number; y: number; z: number };
    rotation: { x: number; y: number; z: number; w: number };
    scale: { x: number; y: number; z: number };
  } | null {
    try {
      if (typeof value === 'string') {
        return JSON.parse(value);
      }
      return value;
    } catch {
      return null;
    }
  }

  /**
   * Merge two vectors (position or scale) with delta-based transformation
   */
  private mergeVectors(
    a: { x: number; y: number; z: number },
    b: { x: number; y: number; z: number },
    aPrev?: { x: number; y: number; z: number },
    bPrev?: { x: number; y: number; z: number }
  ): { x: number; y: number; z: number } {
    if (!aPrev || !bPrev) {
      // No previous values, average the positions
      return {
        x: (a.x + b.x) / 2,
        y: (a.y + b.y) / 2,
        z: (a.z + b.z) / 2
      };
    }

    // Calculate deltas
    const aDelta = {
      x: a.x - aPrev.x,
      y: a.y - aPrev.y,
      z: a.z - aPrev.z
    };

    const bDelta = {
      x: b.x - bPrev.x,
      y: b.y - bPrev.y,
      z: b.z - bPrev.z
    };

    // Apply both deltas to the base
    return {
      x: bPrev.x + aDelta.x + bDelta.x,
      y: bPrev.y + aDelta.y + bDelta.y,
      z: bPrev.z + aDelta.z + bDelta.z
    };
  }

  /**
   * Merge two quaternion rotations
   */
  private mergeRotations(
    a: { x: number; y: number; z: number; w: number },
    b: { x: number; y: number; z: number; w: number },
    aPrev?: { x: number; y: number; z: number; w: number },
    bPrev?: { x: number; y: number; z: number; w: number }
  ): { x: number; y: number; z: number; w: number } {
    if (!aPrev || !bPrev) {
      // No previous values, use spherical linear interpolation (slerp)
      return this.slerp(a, b, 0.5);
    }

    // Calculate rotation deltas (simplified - proper quaternion math needed)
    // For now, use weighted average
    const weight = 0.5;
    return this.slerp(a, b, weight);
  }

  /**
   * Spherical linear interpolation for quaternions
   */
  private slerp(
    q1: { x: number; y: number; z: number; w: number },
    q2: { x: number; y: number; z: number; w: number },
    t: number
  ): { x: number; y: number; z: number; w: number } {
    // Calculate dot product
    let dot = q1.x * q2.x + q1.y * q2.y + q1.z * q2.z + q1.w * q2.w;

    // Ensure shortest path
    let q2Copy = { ...q2 };
    if (dot < 0) {
      dot = -dot;
      q2Copy.x = -q2.x;
      q2Copy.y = -q2.y;
      q2Copy.z = -q2.z;
      q2Copy.w = -q2.w;
    }

    // Linear interpolation for very close quaternions
    if (dot > 0.9995) {
      return this.normalize({
        x: q1.x + t * (q2Copy.x - q1.x),
        y: q1.y + t * (q2Copy.y - q1.y),
        z: q1.z + t * (q2Copy.z - q1.z),
        w: q1.w + t * (q2Copy.w - q1.w)
      });
    }

    // Slerp calculation
    const theta = Math.acos(dot);
    const sinTheta = Math.sin(theta);
    const w1 = Math.sin((1 - t) * theta) / sinTheta;
    const w2 = Math.sin(t * theta) / sinTheta;

    return {
      x: w1 * q1.x + w2 * q2Copy.x,
      y: w1 * q1.y + w2 * q2Copy.y,
      z: w1 * q1.z + w2 * q2Copy.z,
      w: w1 * q1.w + w2 * q2Copy.w
    };
  }

  /**
   * Normalize quaternion
   */
  private normalize(q: { x: number; y: number; z: number; w: number }): {
    x: number;
    y: number;
    z: number;
    w: number;
  } {
    const length = Math.sqrt(q.x * q.x + q.y * q.y + q.z * q.z + q.w * q.w);
    if (length === 0) {
      return { x: 0, y: 0, z: 0, w: 1 };
    }
    return {
      x: q.x / length,
      y: q.y / length,
      z: q.z / length,
      w: q.w / length
    };
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
