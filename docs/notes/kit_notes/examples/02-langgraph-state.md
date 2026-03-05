# Example 02: LangGraph State & Workflow

This example shows how to build a **LangGraph** workflow that manages AI state (drafts, character counts, and safety checks) between different steps.

### 1. State Definition (`graph/state.py`)
```python
from typing import Annotated, List, TypedDict, Union
from langgraph.graph import StateGraph, END

class StoryState(TypedDict):
    """The shared memory of the AI."""
    story_id: str
    prompt: str
    current_draft: str
    is_safe: bool
    iterations: int

def validate_content(state: StoryState):
    """A 'Node' that checks for safety."""
    is_safe = "prohibited" not in state["current_draft"].lower()
    return {"is_safe": is_safe}

def write_story(state: StoryState):
    """A 'Node' that generates text using an LLM."""
    # Imagine calling an LLM here...
    return {"current_draft": f"AI wrote based on: {state['prompt']}", "iterations": state["iterations"] + 1}

def should_continue(state: StoryState):
    """A 'Conditional Edge' to decide next steps."""
    if not state["is_safe"]:
        return "reject"
    if state["iterations"] >= 3:
        return "finish"
    return "rewrite"

# Build the Graph
workflow = StateGraph(StoryState)

# Add Nodes
workflow.add_node("validator", validate_content)
workflow.add_node("writer", write_story)

# Define Edges
workflow.set_entry_point("writer")
workflow.add_edge("writer", "validator")

# Add Conditional Edges
workflow.add_conditional_edges(
    "validator",
    should_continue,
    {
        "rewrite": "writer",
        "finish": END,
        "reject": END
    }
)

# Compile
graph = workflow.compile()
```

### Why use LangGraph?
*   **Cycles:** Standard chains are linear (A -> B -> C). LangGraph allows loops (A -> B -> A).
*   **Persistence:** It can save the "Checkpoint" of a story to a database so you can resume writing later.
*   **Complex Logic:** It manages the state (drafts, character metadata) automatically as it passes between nodes.
