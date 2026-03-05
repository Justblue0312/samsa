# Task 04: LangGraph Foundation (State & Workflow)

## Overview
This task involves setting up the core LangGraph state machine and the base workflow that all agents will follow.

## Objectives
*   Define the global `StoryState` in `kit/graph/state.py`.
*   Setup the base `CompiledGraph` that can route requests to the appropriate agent logic.
*   Integrate LangGraph's streaming capabilities to feed Connect-RPC's `stream` responses.

## Components
*   **State Definition (`kit/graph/state.py`):**
    *   `TypedDict` with fields like `story_id`, `user_id`, `current_draft`, `feedback`, `safety_flags`, `sentiment`.
*   **Base Workflow (`kit/graph/workflow.py`):**
    *   Nodes: `Entry`, `Process`, `Review`, `Exit`.
    *   **[Observability]** Wrap node functions in OTEL spans to track execution time in Jaeger.
    *   Conditional edges based on task type.
*   **Streaming Adapter:** Connect LangGraph's generator output to Connect-RPC's `GenerateStoryResponse` generator.
*   **[Observability]** Record `llm_token_count` metrics for each run.

## Success Criteria
*   A "Hello World" LangGraph workflow can be triggered via RPC.
*   Streaming content works end-to-end (Go -> Python -> Go).
*   State is correctly initialized and updated within the graph nodes.

## Next Steps
*   Implement the `Story Generator Agent` in Task 05.
