"""Configuration settings for story recommender."""

from functools import lru_cache

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings."""

    # Database
    database_url: str = "postgresql+asyncpg://samsa:samsa@localhost:5432/samsa"

    # Redis
    redis_url: str = "redis://localhost:6379"

    # OpenAI
    openai_api_key: str | None = None
    openai_base_url: str = "https://api.openai.com/v1"
    openai_model: str = "gpt-4o-mini"

    # Connect-RPC
    grpc_host: str = "0.0.0.0"
    grpc_port: int = 50051

    # WebSocket
    ws_host: str = "0.0.0.0"
    ws_port: int = 8083

    # Go Server (for callbacks)
    go_server_url: str = "http://localhost:8000"

    # Cache
    cache_ttl: int = 3600  # 1 hour
    cache_max_size: int = 100

    # Agent Limits
    max_tokens_query_generator: int = 4000
    max_requests_query_generator: int = 3
    max_tokens_query_validator: int = 3000
    max_requests_query_validator: int = 2
    max_tokens_recommendation: int = 8000
    max_tool_calls_recommendation: int = 5

    # Freshness decay windows (hours)
    decay_window_bookmarks: int = 48
    decay_window_favorites: int = 72
    decay_window_recently_viewed: int = 24
    decay_window_same_genre: int = 72

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        extra = "ignore"


@lru_cache
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()


settings = get_settings()
