"""SQL Orchestrator - Wires Agent 1 → Checker → Agent 2 → Executor."""

import logging
from dataclasses import dataclass
from typing import Any

from .query_generator import GeneratedQuery, generate_query
from .query_validator import ValidationResult as ValidatorResult
from .query_validator import validate_query
from .sql_validator import ValidationResult, sql_checker

logger = logging.getLogger(__name__)

MAX_CORRECTION_ATTEMPTS = 2


@dataclass
class OrchestratedResult:
    """Result of SQL orchestration."""

    success: bool
    rows: list[dict[str, Any]] | None = None
    error: str | None = None
    query_used: str | None = None


async def orchestrate_sql_pipeline(
    user_id: str,
    context_payload: str,
    db_executor,
) -> OrchestratedResult:
    """
    Execute the full SQL agent pipeline:
    1. Generate SQL (Agent 1)
    2. Deterministic check
    3. Validate/Correct (Agent 2)
    4. Execute (read-only)
    """
    try:
        # Step 1: Generate query
        logger.info("Generating SQL query...")
        generated: GeneratedQuery = await generate_query(
            user_id=user_id,
            context_payload=context_payload,
        )

        sql = generated.sql.strip()
        logger.info(f"Generated SQL: {sql}")

        # Step 2: Deterministic check
        logger.info("Running deterministic check...")
        check_result: ValidationResult = sql_checker.validate(sql, "$1")

        if not check_result.valid:
            logger.warning(f"Deterministic check failed: {check_result.error}")

            # Step 3a: Validate with LLM to try to fix
            for attempt in range(MAX_CORRECTION_ATTEMPTS):
                validator_result: ValidatorResult = await validate_query(
                    user_id=user_id,
                    original_sql=sql,
                    deterministic_error=check_result.error,
                )

                if validator_result.approved:
                    sql = validator_result.sql or sql
                    break
                elif validator_result.corrected_sql:
                    sql = validator_result.corrected_sql
                    # Re-validate corrected SQL
                    check_result = sql_checker.validate(sql, "$1")
                    if check_result.valid:
                        break
                else:
                    return OrchestratedResult(
                        success=False,
                        error=f"Query rejected: {validator_result.rejection_reason}",
                        query_used=sql,
                    )
            else:
                return OrchestratedResult(
                    success=False,
                    error=f"Failed to fix SQL after {MAX_CORRECTION_ATTEMPTS} attempts",
                    query_used=sql,
                )

        # Step 4: Execute query
        logger.info("Executing query...")
        rows = await db_executor.execute_readonly(sql, (user_id,))
        logger.info(f"Query returned {len(rows)} rows")

        return OrchestratedResult(
            success=True,
            rows=rows,
            query_used=sql,
        )

    except Exception as e:
        logger.exception("SQL orchestration failed")
        return OrchestratedResult(
            success=False,
            error=str(e),
        )
