"""Analytics data models"""
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any
from datetime import datetime
from enum import Enum


class Contributor(BaseModel):
    """Session contributor model"""
    user_id: str
    name: str
    operation_count: int
    session_duration: int  # seconds
    last_activity: datetime


class BuildMetrics(BaseModel):
    """Build metrics model"""
    success_rate: float = Field(..., ge=0.0, le=1.0)
    average_build_time: int  # seconds
    total_builds: int
    failed_builds: int


class SessionAnalytics(BaseModel):
    """Session analytics response model"""
    session_id: str
    project_id: str
    duration: int  # seconds
    participant_count: int
    operation_count: int
    top_contributors: List[Contributor]
    objects_created: int
    objects_deleted: int
    objects_modified: int
    average_latency: float  # milliseconds
    peak_memory_usage: int  # MB
    network_bandwidth: int  # KB/s
    build_metrics: Optional[BuildMetrics] = None
    created_at: datetime
    updated_at: datetime


class ProjectAnalytics(BaseModel):
    """Project analytics response model"""
    project_id: str
    total_sessions: int
    total_duration: int  # seconds
    total_participants: int
    unique_contributors: int
    total_operations: int
    average_session_duration: int  # seconds
    collaboration_score: float = Field(..., ge=0.0, le=100.0)
    most_active_users: List[Contributor]
    date_range: Dict[str, str]
    metrics_by_day: List[Dict[str, Any]]


class UserMetrics(BaseModel):
    """User productivity metrics model"""
    user_id: str
    name: str
    total_sessions: int
    total_collaboration_time: int  # seconds
    total_operations: int
    objects_created: int
    objects_modified: int
    objects_deleted: int
    productivity_score: float = Field(..., ge=0.0, le=100.0)
    average_session_duration: int  # seconds
    projects_contributed: List[str]
    last_active: datetime


class ReportType(str, Enum):
    """Custom report types"""
    SESSION_SUMMARY = "session_summary"
    PROJECT_SUMMARY = "project_summary"
    USER_PRODUCTIVITY = "user_productivity"
    PERFORMANCE_TRENDS = "performance_trends"


class CustomReportRequest(BaseModel):
    """Custom report request model"""
    report_type: ReportType
    entity_id: str  # session_id, project_id, or user_id
    start_date: datetime
    end_date: datetime
    filters: Optional[Dict[str, Any]] = None
    aggregation: Optional[str] = "daily"  # hourly, daily, weekly, monthly


class PerformanceMetrics(BaseModel):
    """Performance metrics model"""
    average_latency: float  # milliseconds
    p95_latency: float
    p99_latency: float
    operations_per_second: float
    error_rate: float = Field(..., ge=0.0, le=1.0)
    cpu_usage: float = Field(..., ge=0.0, le=100.0)
    memory_usage: float = Field(..., ge=0.0, le=100.0)


class PerformanceDashboard(BaseModel):
    """Performance dashboard model"""
    active_sessions: int
    active_users: int
    websocket_connections: int
    api_request_rate: float  # requests/second
    sync_service_metrics: PerformanceMetrics
    session_service_metrics: PerformanceMetrics
    asset_service_metrics: PerformanceMetrics
    database_metrics: Dict[str, Any]
    cache_metrics: Dict[str, Any]
    timestamp: datetime
