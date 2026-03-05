"""Schema registry for allowed tables and columns."""

from dataclasses import dataclass


@dataclass
class ColumnInfo:
    """Information about a column."""

    name: str
    type: str
    filterable: bool = False
    selectable: bool = True
    description: str = ""


@dataclass
class TableInfo:
    """Information about a table."""

    name: str
    columns: dict[str, ColumnInfo]
    description: str = ""
    requires_user_filter: bool = False
    max_rows_per_query: int = 100


# Schema registry for story recommendation system
SCHEMA_REGISTRY: dict[str, TableInfo] = {
    "story": TableInfo(
        name="story",
        description="Stories in the system",
        columns={
            "id": ColumnInfo("id", "uuid", filterable=True, selectable=True, description="Story UUID"),
            "owner_id": ColumnInfo("owner_id", "uuid", filterable=True, selectable=True, description="Author UUID"),
            "name": ColumnInfo("name", "text", filterable=True, selectable=True, description="Story title"),
            "slug": ColumnInfo("slug", "text", filterable=True, selectable=True, description="Story slug"),
            "synopsis": ColumnInfo("synopsis", "text", selectable=True, description="Story synopsis"),
            "status": ColumnInfo("status", "text", filterable=True, selectable=True, description="Story status"),
            "is_verified": ColumnInfo("is_verified", "boolean", filterable=True, selectable=True),
            "is_recommended": ColumnInfo("is_recommended", "boolean", filterable=True, selectable=True),
            "first_published_at": ColumnInfo("first_published_at", "timestamp", selectable=True),
            "last_published_at": ColumnInfo("last_published_at", "timestamp", selectable=True),
            "created_at": ColumnInfo("created_at", "timestamp", selectable=True),
            "updated_at": ColumnInfo("updated_at", "timestamp", selectable=True),
        },
    ),
    "chapter": TableInfo(
        name="chapter",
        description="Chapters within stories",
        columns={
            "id": ColumnInfo("id", "uuid", filterable=True, selectable=True),
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "title": ColumnInfo("title", "text", filterable=True, selectable=True),
            "number": ColumnInfo("number", "integer", filterable=True, selectable=True),
            "is_published": ColumnInfo("is_published", "boolean", selectable=True),
            "total_words": ColumnInfo("total_words", "integer", selectable=True),
            "total_views": ColumnInfo("total_views", "integer", selectable=True),
        },
    ),
    "genre": TableInfo(
        name="genre",
        description="Story genres/categories",
        columns={
            "id": ColumnInfo("id", "uuid", filterable=True, selectable=True),
            "name": ColumnInfo("name", "text", filterable=True, selectable=True),
            "description": ColumnInfo("description", "text", selectable=True),
        },
    ),
    "story_genre": TableInfo(
        name="story_genre",
        description="Many-to-many relationship between stories and genres",
        columns={
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "genre_id": ColumnInfo("genre_id", "uuid", filterable=True, selectable=True),
        },
    ),
    "user_bookmark": TableInfo(
        name="user_bookmark",
        description="User bookmarks on stories",
        columns={
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "user_id": ColumnInfo("user_id", "uuid", filterable=True, selectable=True),
            "created_at": ColumnInfo("created_at", "timestamp", selectable=True),
        },
        requires_user_filter=True,
    ),
    "user_favorite": TableInfo(
        name="user_favorite",
        description="User favorites on stories",
        columns={
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "user_id": ColumnInfo("user_id", "uuid", filterable=True, selectable=True),
            "created_at": ColumnInfo("created_at", "timestamp", selectable=True),
        },
        requires_user_filter=True,
    ),
    "story_vote": TableInfo(
        name="story_vote",
        description="User ratings on stories (1-5 stars)",
        columns={
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "user_id": ColumnInfo("user_id", "uuid", filterable=True, selectable=True),
            "rating": ColumnInfo("rating", "integer", filterable=True, selectable=True),
            "created_at": ColumnInfo("created_at", "timestamp", selectable=True),
        },
        requires_user_filter=True,
    ),
    "story_stats_mv": TableInfo(
        name="story_stats_mv",
        description="Aggregated story statistics (materialized view)",
        columns={
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "total_chapters": ColumnInfo("total_chapters", "integer", selectable=True),
            "published_chapters": ColumnInfo("published_chapters", "integer", selectable=True),
            "total_words": ColumnInfo("total_words", "integer", selectable=True),
            "total_views": ColumnInfo("total_views", "integer", selectable=True),
            "total_votes": ColumnInfo("total_votes", "integer", selectable=True),
            "total_favorites": ColumnInfo("total_favorites", "integer", selectable=True),
            "total_bookmarks": ColumnInfo("total_bookmarks", "integer", selectable=True),
        },
    ),
    "story_view_log": TableInfo(
        name="story_view_log",
        description="User story view history",
        columns={
            "story_id": ColumnInfo("story_id", "uuid", filterable=True, selectable=True),
            "user_id": ColumnInfo("user_id", "uuid", filterable=True, selectable=True),
            "viewed_at": ColumnInfo("viewed_at", "timestamp", selectable=True),
        },
        requires_user_filter=True,
    ),
}

# Tables that require user_id filter
TABLES_REQUIRING_USER_FILTER = {name for name, info in SCHEMA_REGISTRY.items() if info.requires_user_filter}

# Forbidden tables (never exposed to agents)
FORBIDDEN_TABLES = {
    "user",
    "session",
    "oauth_account",
    "flag",
    "story_report",
    "submission",
    "submission_assignment",
    "notification",
    "notification_recipient",
}


def get_table_info(table_name: str) -> TableInfo | None:
    """Get table info from registry."""
    return SCHEMA_REGISTRY.get(table_name)


def is_table_allowed(table_name: str) -> bool:
    """Check if table is allowed."""
    return table_name in SCHEMA_REGISTRY


def is_column_allowed(table_name: str, column_name: str) -> bool:
    """Check if column is allowed."""
    table = SCHEMA_REGISTRY.get(table_name)
    if not table:
        return False
    return column_name in table.columns


def is_filterable(table_name: str, column_name: str) -> bool:
    """Check if column can be used in WHERE clause."""
    table = SCHEMA_REGISTRY.get(table_name)
    if not table:
        return False
    col = table.columns.get(column_name)
    return col.filterable if col else False
