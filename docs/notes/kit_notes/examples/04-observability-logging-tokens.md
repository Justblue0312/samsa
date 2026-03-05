# Example 04: Observability - Logging & Token Count

This example shows how to track **AI-specific metrics** (like tokens) and implement **Structured Logging** for Loki.

### 1. Structured Logging (`structlog`)
```python
import structlog
import logging

# Setup structured logging
structlog.configure(
    processors=[
        structlog.processors.JSONRenderer(),  # Output in JSON for Loki
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
    ],
    logger_factory=structlog.PrintLoggerFactory(),
)

log = structlog.get_logger()

async def write_chapter(story_id: str, prompt: str):
    # Log with context
    log.info("start_generation", story_id=story_id, prompt=prompt)
    
    # ... generation logic ...
    
    log.info("complete_generation", story_id=story_id, status="success", duration_ms=450)

# Output in JSON: {"event": "start_generation", "story_id": "123", "prompt": "Write Chapter 1", "level": "info", "timestamp": "2026-02-22T01:52:00Z"}
```

### 2. Token Counting (`tiktoken`)
```python
import tiktoken

def count_tokens(text: str, model="gpt-4o"):
    """Manually count tokens for cost estimation."""
    encoding = tiktoken.encoding_for_model(model)
    return len(encoding.encode(text))

# Usage
prompt_tokens = count_tokens("Once upon a time...")
# Output: 4 tokens
```

### 3. Automatic Tracking (PydanticAI)
```python
from pydantic_ai import Agent, RunContext

agent = Agent("openai:gpt-4o")

async def run_agent():
    result = await agent.run("Hello world!")
    
    # PydanticAI tracks usage automatically!
    print(f"Prompt Tokens: {result.usage.request_tokens}")
    print(f"Completion Tokens: {result.usage.response_tokens}")
    print(f"Total Cost: {result.usage.total_tokens}")

# This is much easier than manual counting!
```

### Why use structured logging?
*   **Context:** Standard `logging.info("Starting task")` is hard to search. `log.info("start_task", user_id=123, story_id=456)` is easy to filter in Grafana Loki to see everything that happened to *just* that user or story.
*   **Tokens:** LLMs are the most expensive part of your system. Tracking tokens per story helps you understand costs and rate-limit users.
