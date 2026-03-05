# Task 06: Agent - Story Analyst (The "Editor")

## Overview
This task involves building the story analyst agent that reviews generated or submitted content for consistency, pacing, and tone.

## Objectives
*   Build an agent that can compare new content against the existing "World Model" (Neo4j).
*   Implement a sentiment and tone analysis node using `pydantic-ai` or standard NLP libraries.
*   Generate a structured report (JSON) for the `AnalyzeStory` RPC response.

## Components
*   **Analyst Node (`kit/agents/analyst.py`):**
    *   `check_consistency(content, story_id)` node: Queries Neo4j for character relationships and Qdrant for past plot points.
    *   `calculate_sentiment(content)` node: Uses an LLM or `vaderSentiment`/`textblob` for scoring.
    *   `analyze_pacing(content)` node: Identifies plot stagnation or rapid shifts.
*   **Report Generator:**
    *   Formats the analyst's findings into the `AnalyzeStoryResponse` proto message.

## Success Criteria
*   The analyst correctly identifies inconsistencies in character relationships (e.g., Character A and B are enemies, but the text treats them as lovers).
*   The sentiment score accurately reflects the emotional tone of the story.
*   The report provides actionable feedback for the user.

## Next Steps
*   Implement the `Content Validator Agent` in Task 07.
