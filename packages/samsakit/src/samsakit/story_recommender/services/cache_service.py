"""Cache Service - Redis caching for recommendations."""

import json
import logging
from datetime import datetime, timezone
from typing import Optional

import redis.asyncio as redis_async

from samsakit.config import settings

logger = logging.getLogger(__name__)


class CacheService:
    """Redis-based caching for recommendations."""

    CACHE_KEY_PREFIX = "samsa:recommendations:"
    CACHE_TTL = settings.cache_ttl  # 1 hour default

    def __init__(self, redis_client: Optional[redis_async.Redis] = None):
        self._redis: Optional[redis_async.Redis] = redis_client
        self._connected = False

    @property
    def redis(self) -> redis_async.Redis:
        """Get Redis client (lazy initialization)."""
        if self._redis is None:
            raise RuntimeError("Redis not initialized. Call set_redis() first.")
        return self._redis

    def set_redis(self, redis_client: redis_async.Redis) -> None:
        """Set the Redis client."""
        self._redis = redis_client
        self._connected = True

    async def set_redis_from_url(self, url: str) -> None:
        """Initialize Redis client from URL."""
        self._redis = redis_async.from_url(
            url,
            encoding="utf-8",
            decode_responses=True,
        )
        await self._redis.ping()  # type: ignore[misc]
        self._connected = True
        logger.info("Redis client initialized")

    async def close(self) -> None:
        """Close the Redis connection."""
        if self._redis:
            await self._redis.aclose()
            logger.info("Redis connection closed")

    async def get_cached_recommendations(
        self,
        user_id: str,
        prompt: str,
    ) -> Optional[list[dict]]:
        """
        Get cached recommendations for a user and prompt.

        Returns None if cache miss or expired.
        """
        try:
            key = self._build_cache_key(user_id, prompt)
            cached = await self.redis.get(key)

            if not cached:
                return None

            data = json.loads(cached)
            logger.info(f"Cache hit for user {user_id}")
            return data.get("recommendations", [])

        except Exception as e:
            logger.warning(f"Cache get failed: {e}")
            return None

    async def cache_recommendations(
        self,
        user_id: str,
        prompt: str,
        story_ids: list[str],
    ) -> bool:
        """
        Cache recommendations for a user and prompt.
        """
        try:
            key = self._build_cache_key(user_id, prompt)
            data = json.dumps(
                {
                    "recommendations": story_ids,
                    "cached_at": datetime.now(timezone.utc).isoformat(),
                }
            )

            await self.redis.setex(key, self.CACHE_TTL, data)
            logger.info(f"Cached {len(story_ids)} recommendations for user {user_id}")
            return True

        except Exception as e:
            logger.warning(f"Cache set failed: {e}")
            return False

    async def invalidate_cache(self, user_id: str) -> bool:
        """Invalidate all cached recommendations for a user."""
        try:
            pattern = f"{self.CACHE_KEY_PREFIX}{user_id}:*"
            keys = await self.redis.keys(pattern)

            if keys:
                await self.redis.delete(*keys)
                logger.info(f"Invalidated {len(keys)} cache entries for user {user_id}")

            return True

        except Exception as e:
            logger.warning(f"Cache invalidation failed: {e}")
            return False

    def _build_cache_key(self, user_id: str, prompt: str) -> str:
        """Build a cache key from user_id and prompt."""
        # Normalize prompt for cache key
        prompt_hash = hash(prompt.lower().strip())
        return f"{self.CACHE_KEY_PREFIX}{user_id}:{prompt_hash}"

    async def get_cache_stats(self) -> dict:
        """Get cache statistics."""
        try:
            info = await self.redis.info("stats")
            keys = await self.redis.keys(f"{self.CACHE_KEY_PREFIX}*")

            return {
                "total_keys": len(keys),
                "hits": info.get("keyspace_hits", 0),
                "misses": info.get("keyspace_misses", 0),
            }

        except Exception as e:
            logger.warning(f"Failed to get cache stats: {e}")
            return {}


async def create_cache_service() -> CacheService:
    """Create and initialize cache service."""
    redis_client = redis_async.from_url(
        settings.redis_url,
        encoding="utf-8",
        decode_responses=True,
    )

    # Test connection
    await redis_client.ping()  # type: ignore[misc]

    service = CacheService(redis_client)
    logger.info("Cache service initialized")
    return service


async def close_cache_service(service: CacheService) -> None:
    """Close cache service."""
    if service._redis:
        await service._redis.aclose()
        logger.info("Cache service closed")
