"""MongoDB database connection management"""
from motor.motor_asyncio import AsyncIOMotorClient
from typing import Optional
import logging

from config import settings

logger = logging.getLogger(__name__)

mongodb_client: Optional[AsyncIOMotorClient] = None


async def get_mongodb_client() -> AsyncIOMotorClient:
    """Get MongoDB client instance"""
    global mongodb_client

    if mongodb_client is None:
        try:
            mongodb_client = AsyncIOMotorClient(
                settings.MONGODB_URI,
                maxPoolSize=50,
                minPoolSize=10
            )

            # Test connection
            await mongodb_client.admin.command('ping')
            logger.info("MongoDB connection established successfully")

            # Create indexes
            await create_indexes()

        except Exception as e:
            logger.error(f"Failed to connect to MongoDB: {e}")
            raise

    return mongodb_client


async def create_indexes():
    """Create MongoDB indexes for analytics collections"""
    try:
        db = mongodb_client[settings.MONGODB_DATABASE]

        # Analytics collection indexes
        await db.analytics.create_index([("session_id", 1), ("timestamp", -1)])
        await db.analytics.create_index([("project_id", 1), ("timestamp", -1)])
        await db.analytics.create_index([("timestamp", -1)])

        # Session analytics indexes
        await db.session_analytics.create_index([("session_id", 1)])
        await db.session_analytics.create_index([("project_id", 1)])

        # User metrics indexes
        await db.user_metrics.create_index([("user_id", 1)])
        await db.user_metrics.create_index([("last_active", -1)])

        logger.info("MongoDB indexes created successfully")

    except Exception as e:
        logger.error(f"Error creating MongoDB indexes: {e}")
        raise


async def close_mongodb_client():
    """Close MongoDB connection"""
    global mongodb_client

    if mongodb_client:
        mongodb_client.close()
        mongodb_client = None
        logger.info("MongoDB connection closed")


def get_database():
    """Get MongoDB database instance"""
    if mongodb_client is None:
        raise Exception("MongoDB client not initialized")

    return mongodb_client[settings.MONGODB_DATABASE]
