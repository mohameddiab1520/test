import jwt from 'jsonwebtoken';
import { config } from '../config';

export interface TokenPayload {
  userId: string;
  email?: string;
  role?: string;
  iat?: number;
  exp?: number;
}

/**
 * Validates a JWT token and returns the decoded payload
 */
export function validateToken(token: string): TokenPayload | null {
  try {
    const decoded = jwt.verify(token, config.jwtSecret) as TokenPayload;
    return decoded;
  } catch (error) {
    console.error('Token validation failed:', error);
    return null;
  }
}

/**
 * Extracts user ID from a JWT token
 */
export function extractUserId(token: string): string | null {
  const payload = validateToken(token);
  return payload ? payload.userId : null;
}

/**
 * Checks if a token is expired
 */
export function isTokenExpired(token: string): boolean {
  try {
    const decoded = jwt.decode(token) as TokenPayload;
    if (!decoded || !decoded.exp) {
      return true;
    }
    return Date.now() >= decoded.exp * 1000;
  } catch {
    return true;
  }
}
