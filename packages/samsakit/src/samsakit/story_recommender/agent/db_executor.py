"""Database executor - Sandboxed read-only execution."""

import logging
from typing import Any

import asyncpg

logger = logging.getLogger(__name__)


class DBExecutor:
    """Sandboxed database executor with read-only transactions."""

    def __init__(self, pool: asyncpg.Pool):
        self.pool = pool

    async def execute_readonly(
        self,
        sql: str,
        parameters: tuple | None = None,
    ) -> list[dict[str, Any]]:
        """
        Execute SQL in a read-only transaction.

        Args:
            sql: SELECT query to execute
            parameters: Query parameters

        Returns:
            List of row dictionaries
        """
        async with self.pool.acquire() as conn:
            # Use read-only transaction
            async with conn.transaction(readonly=True):
                try:
                    params = parameters if parameters is not None else ()
                    rows = await conn.fetch(sql, *params)
                    return [dict(row) for row in rows]
                except Exception as e:
                    logger.error(f"Query execution failed: {e}")
                    logger.error(f"SQL: {sql}")
                    logger.error(f"Params: {parameters}")
                    raise

    async def execute_readonly_single(
        self,
        sql: str,
        parameters: tuple | None = None,
    ) -> dict[str, Any] | None:
        """Execute query and return single row."""
        rows = await self.execute_readonly(sql, parameters)
        return rows[0] if rows else None


async def create_db_executor(database_url: str) -> DBExecutor:
    """Create database executor with connection pool."""
    pool = await asyncpg.create_pool(
        database_url,
        min_size=2,
        max_size=10,
        command_timeout=30,
    )
    logger.info("Database connection pool created")
    return DBExecutor(pool=pool)


async def close_db_executor(executor: DBExecutor) -> None:
    """Close database executor and pool."""
    await executor.pool.close()
    logger.info("Database connection pool closed")
