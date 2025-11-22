"""Redis client connection management"""
import redis.asyncio as redis
from typing import Optional
import logging

from config import settings

logger = logging.getLogger(__name__)

redis_client: Optional[redis.Redis] = None


async def get_redis_client() -> redis.Redis:
    """Get Redis client instance"""
    global redis_client

    if redis_client is None:
        try:
            redis_client = redis.Redis(
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                password=settings.REDIS_PASSWORD,
                db=settings.REDIS_DB,
                decode_responses=True,
                max_connections=50
            )

            # Test connection
            await redis_client.ping()
            logger.info("Redis connection established successfully")

        except Exception as e:
            logger.error(f"Failed to connect to Redis: {e}")
            raise

    return redis_client


async def close_redis_client():
    """Close Redis connection"""
    global redis_client

    if redis_client:
        await redis_client.close()
        redis_client = None
        logger.info("Redis connection closed")
