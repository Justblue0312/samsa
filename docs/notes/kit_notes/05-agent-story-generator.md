# Task 05: Agent - Story Generator (The "Writer")

## Overview
This task involves building the creative writing agent that uses the LLM to generate story content while maintaining world consistency using Neo4j and Qdrant.

## Objectives
*   Implement the `WriteContent` LangGraph node with a real LLM (e.g., GPT-4o, Claude 3.5).
*   Integrate world context from Neo4j (characters, relationships) and Qdrant (past events).
*   Stream the generated text back to the `AgentService.GenerateStory` RPC response.

## Components
*   **LLM Setup (`kit/agents/story_generator.py`):**
    *   Initialize `ChatOpenAI` or `ChatAnthropic` with system prompts for creative writing.
    *   Setup `pydantic-ai` or `langchain` with story-specific tools.
*   **Context Injection Tool:**
    *   `fetch_world_context(story_id, characters)` node.
    *   `query_semantic_memory(prompt)` node.
*   **Review Loop Node:**
    *   `self_review` node to check for tone and style compliance.

## Success Criteria
*   The generator can write a chapter based on a prompt and the existing character graph.
*   The generated text matches the specified story genre and constraints.
*   Relationships from Neo4j are reflected in the story content.

## Next Steps
*   Implement the `Story Analyst Agent` in Task 06.
