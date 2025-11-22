import { Request, Response, NextFunction } from 'express';

/**
 * Error response interface
 */
interface ErrorResponse {
  error: string;
  message: string;
  details?: any;
  timestamp: string;
}

/**
 * Custom error class with status code
 */
export class HttpError extends Error {
  constructor(
    public statusCode: number,
    message: string,
    public details?: any
  ) {
    super(message);
    this.name = 'HttpError';
  }
}

/**
 * Global error handler middleware
 */
export function errorHandler(
  err: Error | HttpError,
  req: Request,
  res: Response,
  next: NextFunction
): void {
  // Default to 500 Internal Server Error
  let statusCode = 500;
  let errorName = 'Internal Server Error';
  let details: any = undefined;

  if (err instanceof HttpError) {
    statusCode = err.statusCode;
    details = err.details;
  } else if (err.name === 'ValidationError') {
    statusCode = 400;
    errorName = 'Validation Error';
  } else if (err.name === 'UnauthorizedError') {
    statusCode = 401;
    errorName = 'Unauthorized';
  }

  const response: ErrorResponse = {
    error: errorName,
    message: err.message,
    timestamp: new Date().toISOString()
  };

  if (details) {
    response.details = details;
  }

  // Don't expose internal errors in production
  if (statusCode === 500 && process.env.NODE_ENV === 'production') {
    response.message = 'An internal error occurred';
  }

  res.status(statusCode).json(response);
}

/**
 * 404 Not Found handler
 */
export function notFoundHandler(req: Request, res: Response): void {
  res.status(404).json({
    error: 'Not Found',
    message: `Route ${req.method} ${req.path} not found`,
    timestamp: new Date().toISOString()
  });
}
