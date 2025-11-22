"""Metrics service for performance monitoring"""
from datetime import datetime
from typing import Dict, Any
import logging
from motor.motor_asyncio import AsyncIOMotorClient
import asyncpg

from models.analytics import PerformanceDashboard, PerformanceMetrics
from config import settings

logger = logging.getLogger(__name__)


class MetricsService:
    """Metrics service for performance monitoring"""

    def __init__(self, mongodb_client: AsyncIOMotorClient, postgres_pool: asyncpg.Pool):
        self.mongodb = mongodb_client[settings.MONGODB_DATABASE]
        self.postgres = postgres_pool

    async def get_performance_dashboard(self) -> PerformanceDashboard:
        """Get performance metrics dashboard"""
        try:
            # Get active sessions and users
            async with self.postgres.acquire() as conn:
                active_sessions = await conn.fetchval(
                    "SELECT COUNT(*) FROM sessions WHERE status = 'active'"
                )

                active_users = await conn.fetchval(
                    """
                    SELECT COUNT(DISTINCT sp.user_id)
                    FROM session_participants sp
                    JOIN sessions s ON sp.session_id = s.session_id
                    WHERE s.status = 'active' AND sp.left_at IS NULL
                    """
                )

            # Get WebSocket connections from Redis
            websocket_connections = 0  # Placeholder

            # Get service metrics (placeholders)
            sync_metrics = PerformanceMetrics(
                average_latency=45.5,
                p95_latency=80.0,
                p99_latency=120.0,
                operations_per_second=1500.0,
                error_rate=0.01,
                cpu_usage=45.0,
                memory_usage=60.0
            )

            session_metrics = PerformanceMetrics(
                average_latency=30.0,
                p95_latency=50.0,
                p99_latency=75.0,
                operations_per_second=500.0,
                error_rate=0.005,
                cpu_usage=35.0,
                memory_usage=50.0
            )

            asset_metrics = PerformanceMetrics(
                average_latency=100.0,
                p95_latency=200.0,
                p99_latency=350.0,
                operations_per_second=200.0,
                error_rate=0.02,
                cpu_usage=40.0,
                memory_usage=55.0
            )

            dashboard = PerformanceDashboard(
                active_sessions=active_sessions,
                active_users=active_users,
                websocket_connections=websocket_connections,
                api_request_rate=2200.0,
                sync_service_metrics=sync_metrics,
                session_service_metrics=session_metrics,
                asset_service_metrics=asset_metrics,
                database_metrics={
                    "postgres": {
                        "connections": 45,
                        "queries_per_second": 300,
                        "average_query_time": 5.2
                    },
                    "redis": {
                        "memory_usage": "512MB",
                        "hit_rate": 0.95
                    },
                    "mongodb": {
                        "connections": 30,
                        "operations_per_second": 150
                    }
                },
                cache_metrics={
                    "hit_rate": 0.85,
                    "miss_rate": 0.15,
                    "eviction_rate": 0.02
                },
                timestamp=datetime.utcnow()
            )

            return dashboard

        except Exception as e:
            logger.error(f"Error getting performance dashboard: {e}")
            raise
