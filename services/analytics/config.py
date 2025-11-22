"""Configuration settings for Analytics Service"""
from pydantic_settings import BaseSettings
from typing import Optional


class Settings(BaseSettings):
    """Application settings"""

    # Service configuration
    SERVICE_NAME: str = "analytics-service"
    SERVICE_PORT: int = 8084
    ENVIRONMENT: str = "development"

    # MongoDB configuration
    MONGODB_URI: str = "mongodb://dev:devpass@localhost:27017"
    MONGODB_DATABASE: str = "collab_analytics"

    # PostgreSQL configuration
    POSTGRES_HOST: str = "localhost"
    POSTGRES_PORT: int = 5432
    POSTGRES_DB: str = "collab_dev"
    POSTGRES_USER: str = "dev"
    POSTGRES_PASSWORD: str = "devpass"
    POSTGRES_POOL_MIN_SIZE: int = 10
    POSTGRES_POOL_MAX_SIZE: int = 20

    # Redis configuration
    REDIS_HOST: str = "localhost"
    REDIS_PORT: int = 6379
    REDIS_PASSWORD: Optional[str] = None
    REDIS_DB: int = 0

    # Analytics configuration
    METRICS_RETENTION_DAYS: int = 30
    AGGREGATION_INTERVAL_MINUTES: int = 1
    CACHE_TTL_SECONDS: int = 300

    # Prometheus configuration
    PROMETHEUS_ENABLED: bool = True
    PROMETHEUS_PORT: int = 9090

    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()
