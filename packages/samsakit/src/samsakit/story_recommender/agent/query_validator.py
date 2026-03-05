"""Query Validator Agent - pydantic-ai Agent 2."""

from pydantic import BaseModel
from pydantic_ai import Agent

from samsakit.config import settings

from .schema_registry import SCHEMA_REGISTRY


class ValidationResult(BaseModel):
    """Result of query validation."""

    approved: bool
    sql: str | None = None
    corrected_sql: str | None = None
    rejection_reason: str | None = None
    changes_made: str | None = None


def get_schema_description() -> str:
    """Generate schema description for prompts."""
    lines = []
    for name, table in SCHEMA_REGISTRY.items():
        lines.append(f"### {name}")
        if table.requires_user_filter:
            lines.append("   ⚠️  REQUIRES user_id filter")
        lines.append(f"   Columns: {', '.join(table.columns.keys())}")
    return "\n".join(lines)


def build_validator_prompt(original_sql: str, user_id: str, error: str | None = None) -> str:
    """Build the prompt for query validator."""
    error_msg = f"Error to fix: {error}\n" if error else ""
    return f"""You are a SQL query validator. Review, correct, or reject queries.

Schema:
{get_schema_description()}

Rules:
- Only SELECT statements allowed
- Must use $1 for user_id parameter
- Tables requiring user_id: user_bookmark, user_favorite, story_vote, story_view_log
- Return JSON with decision

Review this SQL query:

{error_msg}
SQL: {original_sql}

User ID: $1 = {user_id}

If valid, return: {{"approved": true, "sql": "query"}}
If fixable, return: {{"approved": false, "corrected_sql": "fixed", "changes_made": "what"}}
If rejected, return: {{"approved": false, "rejection_reason": "why"}}"""


async def validate_query(
    user_id: str,
    original_sql: str,
    deterministic_error: str | None = None,
) -> ValidationResult:
    """Validate and potentially correct SQL query."""
    agent = Agent(
        model=settings.openai_model,
        output_type=ValidationResult,
    )
    
    prompt = build_validator_prompt(original_sql, user_id, deterministic_error)
    result = await agent.run(prompt)

    return result.output
