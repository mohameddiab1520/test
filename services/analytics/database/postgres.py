"""PostgreSQL database connection management"""
import asyncpg
from typing import Optional
import logging

from config import settings

logger = logging.getLogger(__name__)

postgres_pool: Optional[asyncpg.Pool] = None


async def get_postgres_pool() -> asyncpg.Pool:
    """Get PostgreSQL connection pool"""
    global postgres_pool

    if postgres_pool is None:
        try:
            postgres_pool = await asyncpg.create_pool(
                host=settings.POSTGRES_HOST,
                port=settings.POSTGRES_PORT,
                database=settings.POSTGRES_DB,
                user=settings.POSTGRES_USER,
                password=settings.POSTGRES_PASSWORD,
                min_size=settings.POSTGRES_POOL_MIN_SIZE,
                max_size=settings.POSTGRES_POOL_MAX_SIZE
            )

            logger.info("PostgreSQL connection pool created successfully")

        except Exception as e:
            logger.error(f"Failed to create PostgreSQL connection pool: {e}")
            raise

    return postgres_pool


async def close_postgres_pool():
    """Close PostgreSQL connection pool"""
    global postgres_pool

    if postgres_pool:
        await postgres_pool.close()
        postgres_pool = None
        logger.info("PostgreSQL connection pool closed")
