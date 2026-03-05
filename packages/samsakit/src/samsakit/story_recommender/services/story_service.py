"""Story Service - Database queries for stories."""

import logging
from typing import Any, Optional
from uuid import UUID

import asyncpg

from samsakit.story_recommender.agent.db_executor import DBExecutor

logger = logging.getLogger(__name__)


class StoryService:
    """Service for story-related database operations."""

    def __init__(self, db_executor: Optional[DBExecutor] = None):
        self._db_executor = db_executor

    @property
    def db(self) -> DBExecutor:
        """Get database executor (lazy initialization)."""
        if self._db_executor is None:
            # Will be set by the main app
            from samsakit.database import engine
            from samsakit.story_recommender.agent.db_executor import create_db_executor

            # This is a placeholder - actual initialization happens in main
            raise RuntimeError("DB executor not initialized")
        return self._db_executor

    def set_db_executor(self, executor: DBExecutor):
        """Set the database executor."""
        self._db_executor = executor

    async def get_user_bookmarks(self, user_id: str) -> list[dict]:
        """Get user's bookmarked stories."""
        sql = """
            SELECT 
                s.id, s.name, s.synopsis, s.slug,
                ub.created_at as bookmarked_at
            FROM user_bookmark ub
            JOIN story s ON ub.story_id = s.id
            WHERE ub.user_id = $1
              AND s.status = 'published'
              AND s.is_deleted = FALSE
            ORDER BY ub.created_at DESC
            LIMIT 20
        """
        try:
            return await self.db.execute_readonly(sql, (user_id,))
        except Exception as e:
            logger.error(f"Failed to get bookmarks: {e}")
            return []

    async def get_user_favorites(self, user_id: str) -> list[dict]:
        """Get user's favorited stories."""
        sql = """
            SELECT 
                s.id, s.name, s.synopsis, s.slug,
                uf.created_at as favorited_at
            FROM user_favorite uf
            JOIN story s ON uf.story_id = s.id
            WHERE uf.user_id = $1
              AND s.status = 'published'
              AND s.is_deleted = FALSE
            ORDER BY uf.created_at DESC
            LIMIT 20
        """
        try:
            return await self.db.execute_readonly(sql, (user_id,))
        except Exception as e:
            logger.error(f"Failed to get favorites: {e}")
            return []

    async def get_user_votes(self, user_id: str) -> list[dict]:
        """Get user's story votes (ratings)."""
        sql = """
            SELECT 
                s.id, s.name, s.synopsis, s.slug,
                sv.rating, sv.created_at as voted_at
            FROM story_vote sv
            JOIN story s ON sv.story_id = s.id
            WHERE sv.user_id = $1
              AND s.status = 'published'
              AND s.is_deleted = FALSE
            ORDER BY sv.created_at DESC
            LIMIT 20
        """
        try:
            return await self.db.execute_readonly(sql, (user_id,))
        except Exception as e:
            logger.error(f"Failed to get votes: {e}")
            return []

    async def get_recently_viewed(self, user_id: str) -> list[dict]:
        """Get user's recently viewed stories."""
        sql = """
            SELECT 
                s.id, s.name, s.synopsis, s.slug,
                svl.viewed_at
            FROM story_view_log svl
            JOIN story s ON svl.story_id = s.id
            WHERE svl.user_id = $1
              AND s.status = 'published'
              AND s.is_deleted = FALSE
            ORDER BY svl.viewed_at DESC
            LIMIT 20
        """
        try:
            return await self.db.execute_readonly(sql, (user_id,))
        except Exception as e:
            logger.error(f"Failed to get recently viewed: {e}")
            return []

    async def get_user_genre_preferences(self, user_id: str) -> list[str]:
        """Get user's preferred genres based on favorites/bookmarks."""
        sql = """
            SELECT DISTINCT g.name
            FROM user_favorite uf
            JOIN story_genre sg ON uf.story_id = sg.story_id
            JOIN genre g ON sg.genre_id = g.id
            WHERE uf.user_id = $1
            UNION
            SELECT DISTINCT g.name
            FROM user_bookmark ub
            JOIN story_genre sg ON ub.story_id = sg.story_id
            JOIN genre g ON sg.genre_id = g.id
            WHERE ub.user_id = $1
            LIMIT 10
        """
        try:
            rows = await self.db.execute_readonly(sql, (user_id,))
            return [row["name"] for row in rows]
        except Exception as e:
            logger.error(f"Failed to get genre preferences: {e}")
            return []

    async def get_stories_for_recommendations(
        self,
        user_id: str,
        context,
    ) -> list[dict]:
        """
        Get stories for recommendations using SQL agent pipeline.

        This uses the SQL orchestrator to generate and execute queries.
        """
        from samsakit.story_recommender.agent.sql_orchestrator import orchestrate_sql_pipeline
        from samsakit.story_recommender.agent.db_executor import DBExecutor

        # Create DB executor if not set
        if self._db_executor is None:
            # Get from database module
            from samsakit.database import engine
            import asyncio

            pool = asyncio.get_event_loop().run_until_complete(
                asyncpg.create_pool(
                    str(engine.url),
                    min_size=2,
                    max_size=10,
                )
            )
            self._db_executor = DBExecutor(pool=pool)

        # Build context payload
        context_payload = context.to_prompt()

        # Run SQL orchestration
        result = await orchestrate_sql_pipeline(
            user_id=user_id,
            context_payload=context_payload,
            db_executor=self._db_executor,
        )

        if not result.success:
            logger.warning(f"SQL pipeline failed: {result.error}")
            # Fallback: get popular stories
            return await self._get_popular_stories()

        return result.rows or []

    async def _get_popular_stories(self) -> list[dict]:
        """Fallback: get popular published stories."""
        sql = """
            SELECT 
                s.id, s.name, s.synopsis, s.slug,
                s.owner_id,
                COALESCE(ssm.total_votes, 0) as total_votes,
                COALESCE(ssm.total_views, 0) as total_views,
                COALESCE(ssm.total_favorites, 0) as total_favorites
            FROM story s
            LEFT JOIN story_stats_mv ssm ON s.id = ssm.story_id
            WHERE s.status = 'published'
              AND s.is_deleted = FALSE
            ORDER BY ssm.total_votes DESC, ssm.total_views DESC
            LIMIT 20
        """
        try:
            return await self.db.execute_readonly(sql)
        except Exception as e:
            logger.error(f"Failed to get popular stories: {e}")
            return []

    async def search_stories(
        self,
        keyword: Optional[str] = None,
        genre_id: Optional[str] = None,
        author_id: Optional[str] = None,
        status: str = "published",
        limit: int = 20,
    ) -> list[dict]:
        """Search stories by various filters."""
        conditions = ["s.status = $1", "s.is_deleted = FALSE"]
        params: list[str | int] = [status]
        param_idx = 2

        if keyword:
            conditions.append(f"(s.name ILIKE ${param_idx} OR s.synopsis ILIKE ${param_idx})")
            params.append(f"%{keyword}%")
            param_idx += 1

        if genre_id:
            conditions.append(
                f"EXISTS (SELECT 1 FROM story_genre sg WHERE sg.story_id = s.id AND sg.genre_id = ${param_idx})"
            )
            params.append(genre_id)
            param_idx += 1

        if author_id:
            conditions.append(f"s.owner_id = ${param_idx}")
            params.append(author_id)
            param_idx += 1

        params.append(limit)

        sql = f"""
            SELECT 
                s.id, s.name, s.synopsis, s.slug, s.owner_id,
                s.status, s.is_verified, s.is_recommended,
                s.first_published_at, s.last_published_at,
                COALESCE(ssm.total_votes, 0) as total_votes,
                COALESCE(ssm.total_views, 0) as total_views,
                COALESCE(ssm.total_favorites, 0) as total_favorites,
                COALESCE(ssm.total_chapters, 0) as total_chapters
            FROM story s
            LEFT JOIN story_stats_mv ssm ON s.id = ssm.story_id
            WHERE {" AND ".join(conditions)}
            ORDER BY ssm.total_votes DESC, s.last_published_at DESC
            LIMIT ${param_idx}
        """

        try:
            return await self.db.execute_readonly(sql, tuple(params))
        except Exception as e:
            logger.error(f"Failed to search stories: {e}")
            return []
