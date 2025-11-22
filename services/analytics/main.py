"""
Analytics Service - Main Entry Point
Handles metrics collection, analytics, and reporting for Unity Collaboration Platform
"""
from fastapi import FastAPI, HTTPException, Depends, Query
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from prometheus_client import Counter, Histogram, Gauge, generate_latest
from datetime import datetime, timedelta
from typing import Optional, List
import logging
import asyncio

from config import settings
from database.mongodb import get_mongodb_client, close_mongodb_client
from database.postgres import get_postgres_pool, close_postgres_pool
from database.redis_client import get_redis_client, close_redis_client
from models.analytics import (
    SessionAnalytics,
    ProjectAnalytics,
    UserMetrics,
    CustomReportRequest,
    PerformanceDashboard
)
from services.analytics_service import AnalyticsService
from services.metrics_service import MetricsService

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Unity Collaboration Analytics Service",
    description="Analytics and metrics service for collaboration platform",
    version="1.0.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Prometheus metrics
REQUESTS_TOTAL = Counter(
    'analytics_requests_total',
    'Total number of analytics requests',
    ['method', 'endpoint', 'status']
)
QUERY_DURATION = Histogram(
    'analytics_query_duration_seconds',
    'Duration of analytics queries',
    ['query_type']
)
DATA_POINTS_INGESTED = Counter(
    'analytics_data_points_ingested_total',
    'Total number of data points ingested'
)
ACTIVE_SESSIONS = Gauge(
    'analytics_active_sessions',
    'Number of active collaboration sessions'
)

# Global service instances
analytics_service: Optional[AnalyticsService] = None
metrics_service: Optional[MetricsService] = None


@app.on_event("startup")
async def startup_event():
    """Initialize database connections and services"""
    global analytics_service, metrics_service

    logger.info("Starting Analytics Service...")

    try:
        # Initialize database connections
        mongodb_client = await get_mongodb_client()
        postgres_pool = await get_postgres_pool()
        redis_client = await get_redis_client()

        # Initialize services
        analytics_service = AnalyticsService(
            mongodb_client=mongodb_client,
            postgres_pool=postgres_pool,
            redis_client=redis_client
        )

        metrics_service = MetricsService(
            mongodb_client=mongodb_client,
            postgres_pool=postgres_pool
        )

        logger.info("Analytics Service started successfully")

        # Start background tasks
        asyncio.create_task(collect_active_sessions_metric())

    except Exception as e:
        logger.error(f"Failed to start Analytics Service: {e}")
        raise


@app.on_event("shutdown")
async def shutdown_event():
    """Close database connections"""
    logger.info("Shutting down Analytics Service...")

    await close_mongodb_client()
    await close_postgres_pool()
    await close_redis_client()

    logger.info("Analytics Service shut down successfully")


async def collect_active_sessions_metric():
    """Background task to collect active sessions metric"""
    while True:
        try:
            if analytics_service:
                active_count = await analytics_service.get_active_sessions_count()
                ACTIVE_SESSIONS.set(active_count)
        except Exception as e:
            logger.error(f"Error collecting active sessions metric: {e}")

        await asyncio.sleep(15)  # Update every 15 seconds


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "analytics", "timestamp": datetime.utcnow().isoformat()}


@app.get("/metrics")
async def metrics():
    """Prometheus metrics endpoint"""
    return generate_latest()


@app.get("/api/v1/analytics/sessions/{session_id}", response_model=SessionAnalytics)
async def get_session_analytics(session_id: str):
    """Get analytics for a specific session"""
    try:
        with QUERY_DURATION.labels(query_type='session').time():
            analytics = await analytics_service.get_session_analytics(session_id)

        REQUESTS_TOTAL.labels(method='GET', endpoint='/sessions', status='success').inc()
        return analytics

    except Exception as e:
        logger.error(f"Error getting session analytics: {e}")
        REQUESTS_TOTAL.labels(method='GET', endpoint='/sessions', status='error').inc()
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/v1/analytics/projects/{project_id}", response_model=ProjectAnalytics)
async def get_project_analytics(
    project_id: str,
    from_date: Optional[str] = Query(None, description="Start date (ISO format)"),
    to_date: Optional[str] = Query(None, description="End date (ISO format)")
):
    """Get analytics for a project with optional date range"""
    try:
        # Parse dates
        start_date = datetime.fromisoformat(from_date) if from_date else datetime.utcnow() - timedelta(days=30)
        end_date = datetime.fromisoformat(to_date) if to_date else datetime.utcnow()

        with QUERY_DURATION.labels(query_type='project').time():
            analytics = await analytics_service.get_project_analytics(
                project_id,
                start_date,
                end_date
            )

        REQUESTS_TOTAL.labels(method='GET', endpoint='/projects', status='success').inc()
        return analytics

    except Exception as e:
        logger.error(f"Error getting project analytics: {e}")
        REQUESTS_TOTAL.labels(method='GET', endpoint='/projects', status='error').inc()
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/v1/analytics/users/{user_id}/metrics", response_model=UserMetrics)
async def get_user_metrics(user_id: str):
    """Get productivity metrics for a user"""
    try:
        with QUERY_DURATION.labels(query_type='user').time():
            metrics = await analytics_service.get_user_metrics(user_id)

        REQUESTS_TOTAL.labels(method='GET', endpoint='/users/metrics', status='success').inc()
        return metrics

    except Exception as e:
        logger.error(f"Error getting user metrics: {e}")
        REQUESTS_TOTAL.labels(method='GET', endpoint='/users/metrics', status='error').inc()
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/v1/analytics/custom-report")
async def create_custom_report(report_request: CustomReportRequest):
    """Create a custom analytics report"""
    try:
        with QUERY_DURATION.labels(query_type='custom_report').time():
            report = await analytics_service.generate_custom_report(report_request)

        REQUESTS_TOTAL.labels(method='POST', endpoint='/custom-report', status='success').inc()
        return report

    except Exception as e:
        logger.error(f"Error creating custom report: {e}")
        REQUESTS_TOTAL.labels(method='POST', endpoint='/custom-report', status='error').inc()
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/v1/analytics/performance/dashboard", response_model=PerformanceDashboard)
async def get_performance_dashboard():
    """Get performance metrics dashboard"""
    try:
        with QUERY_DURATION.labels(query_type='dashboard').time():
            dashboard = await metrics_service.get_performance_dashboard()

        REQUESTS_TOTAL.labels(method='GET', endpoint='/performance/dashboard', status='success').inc()
        return dashboard

    except Exception as e:
        logger.error(f"Error getting performance dashboard: {e}")
        REQUESTS_TOTAL.labels(method='GET', endpoint='/performance/dashboard', status='error').inc()
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/api/v1/analytics/ingest")
async def ingest_metrics(metrics_data: dict):
    """Ingest metrics data from other services"""
    try:
        await analytics_service.ingest_metrics(metrics_data)
        DATA_POINTS_INGESTED.inc()

        REQUESTS_TOTAL.labels(method='POST', endpoint='/ingest', status='success').inc()
        return {"status": "success", "message": "Metrics ingested successfully"}

    except Exception as e:
        logger.error(f"Error ingesting metrics: {e}")
        REQUESTS_TOTAL.labels(method='POST', endpoint='/ingest', status='error').inc()
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8084)
