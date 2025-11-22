import { Request, Response, NextFunction } from 'express';

/**
 * Request logging middleware
 */
export function requestLogger(req: Request, res: Response, next: NextFunction): void {
  const start = Date.now();

  // Log request
  console.log(`--> ${req.method} ${req.path}`, {
    ip: req.ip,
    userAgent: req.get('user-agent'),
    timestamp: new Date().toISOString()
  });

  // Capture response
  res.on('finish', () => {
    const duration = Date.now() - start;
    const logLevel = res.statusCode >= 400 ? 'error' : 'info';

    const message = `<-- ${req.method} ${req.path} ${res.statusCode} ${duration}ms`;

    if (logLevel === 'error') {
      console.error(message);
    } else {
      console.log(message);
    }
  });

  next();
}

/**
 * Error logging middleware
 */
export function errorLogger(
  err: Error,
  req: Request,
  res: Response,
  next: NextFunction
): void {
  console.error('Error:', {
    message: err.message,
    stack: err.stack,
    path: req.path,
    method: req.method,
    timestamp: new Date().toISOString()
  });

  next(err);
}
