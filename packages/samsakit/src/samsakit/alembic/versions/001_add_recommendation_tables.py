"""Add story_view_log and recommendation_cache tables.

Revision ID: 001_add_recommendation_tables
Revises:
Create Date: 2026-03-05 10:00:00.000000

"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision: str = "001_add_recommendation_tables"
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # Create story_view_log table
    op.create_table(
        "story_view_log",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("story_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("user_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("viewed_at", sa.DateTime(timezone=True), nullable=False),
    )

    # Create indexes for story_view_log
    op.create_index("idx_story_view_log_user_id", "story_view_log", ["user_id"])
    op.create_index("idx_story_view_log_story_id", "story_view_log", ["story_id"])
    op.create_index("idx_story_view_log_viewed_at", "story_view_log", ["viewed_at"])

    # Create recommendation_cache table
    op.create_table(
        "recommendation_cache",
        sa.Column("user_id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("story_ids", postgresql.ARRAY(postgresql.UUID(as_uuid=True)), nullable=False),
        sa.Column("prompt", sa.Text(), nullable=True),
        sa.Column("cached_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("expires_at", sa.DateTime(timezone=True), nullable=True),
    )


def downgrade() -> None:
    op.drop_table("recommendation_cache")
    op.drop_index("idx_story_view_log_viewed_at")
    op.drop_index("idx_story_view_log_story_id")
    op.drop_index("idx_story_view_log_user_id")
    op.drop_table("story_view_log")
