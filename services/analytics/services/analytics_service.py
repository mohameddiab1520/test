"""Analytics service business logic"""
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
import logging
from motor.motor_asyncio import AsyncIOMotorClient
import asyncpg

from models.analytics import (
    SessionAnalytics,
    ProjectAnalytics,
    UserMetrics,
    CustomReportRequest,
    Contributor,
    BuildMetrics
)
from config import settings

logger = logging.getLogger(__name__)


class AnalyticsService:
    """Analytics service for metrics collection and analysis"""

    def __init__(
        self,
        mongodb_client: AsyncIOMotorClient,
        postgres_pool: asyncpg.Pool,
        redis_client
    ):
        self.mongodb = mongodb_client[settings.MONGODB_DATABASE]
        self.postgres = postgres_pool
        self.redis = redis_client

    async def get_session_analytics(self, session_id: str) -> SessionAnalytics:
        """Get analytics for a specific session"""
        try:
            # Check cache first
            cached = await self.redis.get(f"analytics:session:{session_id}")
            if cached:
                import json
                return SessionAnalytics(**json.loads(cached))

            # Query MongoDB for session analytics
            analytics_doc = await self.mongodb.session_analytics.find_one(
                {"session_id": session_id}
            )

            if not analytics_doc:
                # Generate analytics from operations
                analytics_doc = await self._generate_session_analytics(session_id)

            # Convert to model
            analytics = SessionAnalytics(**analytics_doc)

            # Cache result
            import json
            await self.redis.setex(
                f"analytics:session:{session_id}",
                settings.CACHE_TTL_SECONDS,
                json.dumps(analytics.model_dump(mode='json'), default=str)
            )

            return analytics

        except Exception as e:
            logger.error(f"Error getting session analytics: {e}")
            raise

    async def _generate_session_analytics(self, session_id: str) -> Dict[str, Any]:
        """Generate analytics from session data"""
        try:
            # Get session info from PostgreSQL
            async with self.postgres.acquire() as conn:
                session = await conn.fetchrow(
                    """
                    SELECT s.*, p.name as project_name
                    FROM sessions s
                    LEFT JOIN projects p ON s.project_id = p.project_id
                    WHERE s.session_id = $1
                    """,
                    session_id
                )

                if not session:
                    raise Exception(f"Session {session_id} not found")

                # Get participant stats
                participants = await conn.fetch(
                    """
                    SELECT
                        sp.user_id,
                        u.name,
                        COUNT(*) as operation_count,
                        EXTRACT(EPOCH FROM (sp.left_at - sp.joined_at)) as session_duration,
                        MAX(sp.left_at) as last_activity
                    FROM session_participants sp
                    LEFT JOIN users u ON sp.user_id = u.user_id
                    WHERE sp.session_id = $1
                    GROUP BY sp.user_id, u.name, sp.joined_at, sp.left_at
                    ORDER BY operation_count DESC
                    """,
                    session_id
                )

            # Get operations from MongoDB
            operations = await self.mongodb.analytics.find(
                {"session_id": session_id}
            ).to_list(length=None)

            # Calculate metrics
            total_operations = len(operations)
            objects_created = sum(1 for op in operations if op.get("type") == "create")
            objects_deleted = sum(1 for op in operations if op.get("type") == "delete")
            objects_modified = sum(1 for op in operations if op.get("type") == "update")

            latencies = [op.get("latency", 0) for op in operations if "latency" in op]
            avg_latency = sum(latencies) / len(latencies) if latencies else 0

            # Build contributors list
            contributors = [
                {
                    "user_id": p["user_id"],
                    "name": p["name"],
                    "operation_count": p["operation_count"],
                    "session_duration": int(p["session_duration"] or 0),
                    "last_activity": p["last_activity"] or datetime.utcnow()
                }
                for p in participants
            ]

            analytics = {
                "session_id": session_id,
                "project_id": session["project_id"],
                "duration": int((session["updated_at"] - session["created_at"]).total_seconds()),
                "participant_count": len(participants),
                "operation_count": total_operations,
                "top_contributors": contributors[:5],
                "objects_created": objects_created,
                "objects_deleted": objects_deleted,
                "objects_modified": objects_modified,
                "average_latency": avg_latency,
                "peak_memory_usage": 0,  # Placeholder
                "network_bandwidth": 0,  # Placeholder
                "build_metrics": None,
                "created_at": session["created_at"],
                "updated_at": session["updated_at"]
            }

            # Store in MongoDB
            await self.mongodb.session_analytics.update_one(
                {"session_id": session_id},
                {"$set": analytics},
                upsert=True
            )

            return analytics

        except Exception as e:
            logger.error(f"Error generating session analytics: {e}")
            raise

    async def get_project_analytics(
        self,
        project_id: str,
        start_date: datetime,
        end_date: datetime
    ) -> ProjectAnalytics:
        """Get analytics for a project"""
        try:
            async with self.postgres.acquire() as conn:
                # Get project sessions
                sessions = await conn.fetch(
                    """
                    SELECT *
                    FROM sessions
                    WHERE project_id = $1
                      AND created_at BETWEEN $2 AND $3
                    """,
                    project_id,
                    start_date,
                    end_date
                )

                # Get unique contributors
                contributors = await conn.fetch(
                    """
                    SELECT DISTINCT
                        u.user_id,
                        u.name,
                        COUNT(*) as operation_count,
                        SUM(EXTRACT(EPOCH FROM (sp.left_at - sp.joined_at))) as total_time
                    FROM session_participants sp
                    JOIN users u ON sp.user_id = u.user_id
                    JOIN sessions s ON sp.session_id = s.session_id
                    WHERE s.project_id = $1
                      AND s.created_at BETWEEN $2 AND $3
                    GROUP BY u.user_id, u.name
                    ORDER BY operation_count DESC
                    """,
                    project_id,
                    start_date,
                    end_date
                )

            # Calculate metrics
            total_sessions = len(sessions)
            total_duration = sum(
                int((s["updated_at"] - s["created_at"]).total_seconds())
                for s in sessions
            )
            avg_duration = total_duration // total_sessions if total_sessions > 0 else 0

            # Build contributors list
            top_contributors = [
                {
                    "user_id": c["user_id"],
                    "name": c["name"],
                    "operation_count": c["operation_count"],
                    "session_duration": int(c["total_time"] or 0),
                    "last_activity": datetime.utcnow()
                }
                for c in contributors[:10]
            ]

            analytics = ProjectAnalytics(
                project_id=project_id,
                total_sessions=total_sessions,
                total_duration=total_duration,
                total_participants=len(contributors),
                unique_contributors=len(contributors),
                total_operations=0,  # Placeholder
                average_session_duration=avg_duration,
                collaboration_score=75.0,  # Placeholder
                most_active_users=top_contributors,
                date_range={
                    "start": start_date.isoformat(),
                    "end": end_date.isoformat()
                },
                metrics_by_day=[]
            )

            return analytics

        except Exception as e:
            logger.error(f"Error getting project analytics: {e}")
            raise

    async def get_user_metrics(self, user_id: str) -> UserMetrics:
        """Get productivity metrics for a user"""
        try:
            async with self.postgres.acquire() as conn:
                # Get user info
                user = await conn.fetchrow(
                    "SELECT * FROM users WHERE user_id = $1",
                    user_id
                )

                if not user:
                    raise Exception(f"User {user_id} not found")

                # Get user sessions
                sessions = await conn.fetch(
                    """
                    SELECT s.*
                    FROM sessions s
                    JOIN session_participants sp ON s.session_id = sp.session_id
                    WHERE sp.user_id = $1
                    """,
                    user_id
                )

                # Get projects contributed to
                projects = await conn.fetch(
                    """
                    SELECT DISTINCT s.project_id
                    FROM sessions s
                    JOIN session_participants sp ON s.session_id = sp.session_id
                    WHERE sp.user_id = $1
                    """,
                    user_id
                )

            # Calculate metrics
            total_sessions = len(sessions)
            total_time = sum(
                int((s["updated_at"] - s["created_at"]).total_seconds())
                for s in sessions
            )
            avg_duration = total_time // total_sessions if total_sessions > 0 else 0

            metrics = UserMetrics(
                user_id=user_id,
                name=user["name"],
                total_sessions=total_sessions,
                total_collaboration_time=total_time,
                total_operations=0,  # Placeholder
                objects_created=0,  # Placeholder
                objects_modified=0,  # Placeholder
                objects_deleted=0,  # Placeholder
                productivity_score=80.0,  # Placeholder
                average_session_duration=avg_duration,
                projects_contributed=[p["project_id"] for p in projects],
                last_active=user.get("last_login_at", datetime.utcnow())
            )

            return metrics

        except Exception as e:
            logger.error(f"Error getting user metrics: {e}")
            raise

    async def generate_custom_report(self, request: CustomReportRequest) -> Dict[str, Any]:
        """Generate custom analytics report"""
        try:
            if request.report_type == "session_summary":
                analytics = await self.get_session_analytics(request.entity_id)
                return analytics.model_dump()

            elif request.report_type == "project_summary":
                analytics = await self.get_project_analytics(
                    request.entity_id,
                    request.start_date,
                    request.end_date
                )
                return analytics.model_dump()

            elif request.report_type == "user_productivity":
                metrics = await self.get_user_metrics(request.entity_id)
                return metrics.model_dump()

            else:
                raise Exception(f"Unknown report type: {request.report_type}")

        except Exception as e:
            logger.error(f"Error generating custom report: {e}")
            raise

    async def ingest_metrics(self, metrics_data: Dict[str, Any]) -> None:
        """Ingest metrics data from other services"""
        try:
            # Add timestamp if not present
            if "timestamp" not in metrics_data:
                metrics_data["timestamp"] = datetime.utcnow()

            # Store in MongoDB
            await self.mongodb.analytics.insert_one(metrics_data)

            logger.info(f"Ingested metrics: {metrics_data.get('type', 'unknown')}")

        except Exception as e:
            logger.error(f"Error ingesting metrics: {e}")
            raise

    async def get_active_sessions_count(self) -> int:
        """Get count of active sessions"""
        try:
            async with self.postgres.acquire() as conn:
                count = await conn.fetchval(
                    """
                    SELECT COUNT(*)
                    FROM sessions
                    WHERE status = 'active'
                    """
                )
            return count

        except Exception as e:
            logger.error(f"Error getting active sessions count: {e}")
            return 0
