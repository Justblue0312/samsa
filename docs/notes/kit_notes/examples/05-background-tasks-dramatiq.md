# Example 05: Background Tasks - Dramatiq

This example shows how to use **Dramatiq** to handle long-running AI tasks (like generating a 5,000-word story) without blocking your API server.

### 1. Broker Setup (`kit/worker.py`)
```python
import dramatiq
from dramatiq.brokers.redis import RedisBroker

# Connect to Redis (from your docker-compose)
redis_broker = RedisBroker(host="redis", port=6379)
dramatiq.set_broker(redis_broker)

@dramatiq.actor
def process_long_story(story_id: str, prompt: str):
    """A background worker that runs AI logic."""
    # 1. Start generation...
    # 2. Update Postgres database with progress...
    # 3. Complete generation.
    print(f"Working on story {story_id} in background...")

# To start the worker: `dramatiq kit.worker`
```

### 2. Triggering Task from API (`kit/api/server.py`)
```python
from kit.worker import process_long_story

@app.post("/stories/{story_id}/generate")
async def trigger_generation(story_id: str, prompt: str):
    """API endpoint triggers the background task."""
    
    # Send task to Redis queue
    process_long_story.send(story_id, prompt)
    
    # Return 202 Accepted immediately
    return {"status": "accepted", "task_id": story_id}
```

### Why use Dramatiq?
*   **Timeouts:** LLM generation can take 60+ seconds. Most API gateways (like Nginx) or clients will timeout.
*   **Reliability:** If the Python process crashes, Redis "remembers" the task and will retry it automatically.
*   **Scalability:** You can run 1 API server but 10 background worker servers to handle many stories at once.

### Comparison
*   **FastAPI / Connect-RPC:** Fast, interactive, real-time streaming.
*   **Dramatiq:** Slow, background, "Fire and Forget" tasks.
