"""Query Generator Agent - pydantic-ai Agent 1."""

from pydantic import BaseModel
from pydantic_ai import Agent

from samsakit.config import settings

from .schema_registry import SCHEMA_REGISTRY


class GeneratedQuery(BaseModel):
    """Generated SQL query with metadata."""

    sql: str
    intent: str
    tables_used: list[str]


def get_schema_description() -> str:
    """Generate schema description for prompts."""
    lines = []
    for name, table in SCHEMA_REGISTRY.items():
        lines.append(f"### {name}")
        lines.append(f"   Description: {table.description}")
        if table.requires_user_filter:
            lines.append("   ⚠️  REQUIRES user_id filter in WHERE clause")
        lines.append("   Columns:")
        for col_name, col in table.columns.items():
            flags = []
            if col.filterable:
                flags.append("filterable")
            if col.selectable:
                flags.append("selectable")
            flag_str = f" [{', '.join(flags)}]" if flags else ""
            lines.append(f"      - {col_name} ({col.type}){flag_str}")
        lines.append("")
    return "\n".join(lines)


def build_query_generator_prompt(context_payload: str) -> str:
    """Build the prompt for query generator."""
    return f"""You are a SQL query generator for a story recommendation system.
Generate safe, efficient SELECT queries based on user context.

Schema:
{get_schema_description()}

Important:
- Only use tables and columns from the schema
- Tables requiring user_id filter: user_bookmark, user_favorite, story_vote, story_view_log
- Always use parameterized queries ($1 for user_id)
- Return only the SQL, no explanations

User Context:
{context_payload}

Generate a single SELECT query that:
1. Finds stories matching the user's interests
2. Uses appropriate JOINs with related tables (bookmarks, favorites, votes, genres)
3. Orders by relevance (popularity, ratings, recency)
4. Limits to 10 results

Return the SQL query only."""


async def generate_query(
    user_id: str,
    context_payload: str,
) -> GeneratedQuery:
    """Generate SQL query using the agent."""
    agent = Agent(
        model=settings.openai_model,
        output_type=GeneratedQuery,
    )
    
    prompt = build_query_generator_prompt(context_payload)
    result = await agent.run(prompt)

    return result.output
