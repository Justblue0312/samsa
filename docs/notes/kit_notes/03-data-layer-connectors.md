# Task 03: Database Connectors (Postgres, Neo4j, Qdrant)

## Overview
This task involves setting up asynchronous database connectors in the `kit` project so agents can access character relationships, story metadata, and semantic memory.

## Objectives
*   Configure `sqlalchemy` with `asyncpg` for Postgres access.
*   Configure the async `neo4j` driver for the character graph.
*   Configure the async `qdrant-client` for semantic memory.

## Components
*   **Postgres Connector (`kit/services/postgres.py`):**
    *   Setup `async_sessionmaker`.
    *   **[Observability]** Use `SQLAlchemyInstrumentor` to trace database queries.
    *   Create a simple `get_story_context(story_id)` utility to fetch story metadata.
*   **Neo4j Connector (`kit/services/neo4j.py`):**
    *   Async driver initialization.
    *   **[Observability]** Add custom spans for character relationship lookups.
    *   `get_character_relationships(char_a, char_b)` utility.
*   **Qdrant Connector (`kit/services/qdrant.py`):**
    *   Async client initialization.
    *   **[Observability]** Add custom spans for semantic memory searches.
    *   `search_memory(query)` and `add_memory(content)` utilities.
*   **Dependency Injection:** Ensure these connectors can be injected into LangGraph nodes.

## Success Criteria
*   Unit tests for each connector successfully read/write to the databases.
*   Connection pooling is configured and optimized for async workloads.
*   No blocking database calls in the main event loop.

## Next Steps
*   Setup the LangGraph state machine in Task 04.
