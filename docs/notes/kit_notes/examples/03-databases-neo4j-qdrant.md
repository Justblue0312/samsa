# Example 03: Databases - Neo4j & Qdrant

This example shows how to use **Neo4j** (character graphs) and **Qdrant** (semantic memory) to give the AI context.

### 1. Neo4j (Graph Database)
```python
import asyncio
from neo4j import AsyncGraphDatabase

async def get_character_relationships(char_name: str):
    """How is this character related to others?"""
    async with AsyncGraphDatabase.driver("bolt://localhost:7687", auth=("neo4j", "password")) as driver:
        async with driver.session() as session:
            # Cypher query to find direct relationships
            result = await session.run(
                "MATCH (c1:Character {name: $name})-[r]->(c2:Character) "
                "RETURN c2.name, type(r) as relationship",
                name=char_name
            )
            return [f"{record['c2.name']} is {record['relationship']}" for record in result]

# Output might be: ["Sarah is Enemies", "Bob is Friends"]
```

### 2. Qdrant (Vector Database)
```python
from qdrant_client import AsyncQdrantClient
from qdrant_client.models import Distance, VectorParams

# Initialize client
client = AsyncQdrantClient(host="localhost", port=6333)

async def search_story_memory(story_id: str, query: str):
    """What happened in this story before?"""
    # Search for similar plot points or events
    results = await client.search(
        collection_name="story_memory",
        query_vector=[0.1, 0.2, 0.3, ...],  # This is the 'embedding' of your query
        query_filter={"must": [{"key": "story_id", "match": {"value": story_id}}]},
        limit=3
    )
    return [hit.payload["text"] for hit in results]

# Output might be: ["In Chapter 1, Bob lost his sword.", "In Chapter 2, Bob found a shield."]
```

### Why use both?
*   **Neo4j:** Perfect for "Fixed Facts" (e.g., A is B's brother). Computers are better at this than LLMs.
*   **Qdrant:** Perfect for "Vague Memories" (e.g., Search for 'When did Bob mention he was hungry?'). It allows the AI to 'recall' context from 100 pages ago without reading the whole book.
