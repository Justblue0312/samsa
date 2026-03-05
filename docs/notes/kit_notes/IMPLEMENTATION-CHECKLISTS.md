# Kit Implementation Checklists

Use these checklists to track progress on each task. Mark items as `[x]` when complete.

---

## Task 01: Proto Definition & Code Generation

**Goal:** Define communication contract between Go Server and Python AI Toolkit.

### Setup
- [ ] Install `buf` CLI tool
- [ ] Create `kit/v1/` directory structure
- [ ] Configure `buf.yaml` in project root
- [ ] Configure `buf.gen.yaml` with Go + Python plugins

### Proto Definition
- [ ] Create `kit/v1/service.proto`
- [ ] Define `StoryContext` message
- [ ] Define `GenerateStory` RPC (server-streaming)
- [ ] Define `AnalyzeStory` RPC (unary)
- [ ] Define `ValidateContent` RPC (unary)
- [ ] Define `SubmissionApprove` RPC (unary)
- [ ] Add all request/response message types

### Code Generation
- [ ] Run `buf generate` successfully
- [ ] Verify Python stubs in `kit/src/kit/gen/`
- [ ] Verify Go stubs in `server/internal/gen/`
- [ ] Create test `main.go` importing `AgentServiceClient`
- [ ] Confirm no compile errors in Go

### Documentation
- [ ] Document proto message fields
- [ ] Add comments for each RPC method
- [ ] Update `buf.gen.yaml` if plugins need adjustment

---

## Task 02: Python Service Setup (FastAPI + Connect-RPC)

**Goal:** Bootstrap FastAPI server hosting AI Agents with Connect-RPC.

### Dependencies
- [ ] Add `connect-rpc-python` to `pyproject.toml`
- [ ] Add `structlog` for structured logging
- [ ] Add `uvicorn` (if not present)
- [ ] Run `uv sync` to install dependencies

### Server Setup
- [ ] Create `kit/api/server.py`
- [ ] Initialize FastAPI application
- [ ] Add `/health` endpoint
- [ ] Configure CORS middleware (if needed)
- [ ] Setup graceful shutdown handling

### Connect-RPC Integration
- [ ] Mount `AgentService` handler using Connect-RPC
- [ ] Create `kit/api/handlers.py`
- [ ] Implement `AgentService` class skeleton
- [ ] Implement `AnalyzeStory` with mocked response
- [ ] Implement `ValidateContent` with mocked response
- [ ] Implement `GenerateStory` with streaming mock

### Configuration
- [ ] Extend `kit/settings.py` for environment variables
- [ ] Add `SERVER_HOST`, `SERVER_PORT` settings
- [ ] Add `OTEL_EXPORTER_OTLP_ENDPOINT` setting
- [ ] Load settings from `.env` file

### Logging
- [ ] Configure `structlog` with JSON renderer
- [ ] Add request/response logging middleware
- [ ] Test log output format compatibility with Loki

### Verification
- [ ] Server starts via `uv run kit`
- [ ] `curl` to `/kit.v1.AgentService/AnalyzeStory` returns response
- [ ] Go client can call Python server successfully
- [ ] Logs appear in JSON format

---

## Task 03: Database Connectors (Postgres, Neo4j, Qdrant)

**Goal:** Setup async database connectors for agent data access.

### Postgres (SQLAlchemy + asyncpg)
- [ ] Add `asyncpg` to dependencies (already present)
- [ ] Create `kit/services/postgres.py`
- [ ] Initialize async engine with connection pool
- [ ] Create `async_sessionmaker`
- [ ] Implement `get_story_context(story_id)` utility
- [ ] Implement `get_user_reputation(user_id)` utility
- [ ] Add dependency injection pattern for sessions
- [ ] Write unit tests for read/write operations

### Neo4j (Graph Database)
- [ ] Add `neo4j` async driver to dependencies
- [ ] Create `kit/services/neo4j.py`
- [ ] Initialize async driver with auth
- [ ] Implement `get_character_relationships(char_a, char_b)`
- [ ] Implement `get_story_graph(story_id)` utility
- [ ] Add connection pool configuration
- [ ] Write unit tests for relationship queries

### Qdrant (Vector Database)
- [ ] Add `qdrant-client` to dependencies
- [ ] Create `kit/services/qdrant.py`
- [ ] Initialize async Qdrant client
- [ ] Implement `search_memory(query, story_id, limit)`
- [ ] Implement `add_memory(content, story_id, embedding)` utility
- [ ] Configure collection creation if not exists
- [ ] Write unit tests for search/add operations

### Dependency Injection
- [ ] Create `kit/services/__init__.py` with exports
- [ ] Ensure connectors can be injected into LangGraph nodes
- [ ] Create service container/factory pattern (optional)

### Verification
- [ ] All connectors initialize without errors
- [ ] Unit tests pass for each connector
- [ ] No blocking calls in async event loop
- [ ] Connection pooling configured correctly

---

## Task 04: LangGraph Foundation (State & Workflow)

**Goal:** Setup core LangGraph state machine and base workflow.

### State Definition
- [ ] Create `kit/graph/state.py`
- [ ] Define `StoryState` TypedDict with:
  - [ ] `story_id: str`
  - [ ] `user_id: str`
  - [ ] `current_draft: str`
  - [ ] `feedback: str`
  - [ ] `safety_flags: list`
  - [ ] `sentiment: dict`
  - [ ] `iterations: int`
- [ ] Add type hints for all fields
- [ ] Document state field purposes

### Base Workflow
- [ ] Create `kit/graph/workflow.py`
- [ ] Initialize `StateGraph` with `StoryState`
- [ ] Create `Entry` node (state initialization)
- [ ] Create `Process` node (routing logic)
- [ ] Create `Review` node (aggregation)
- [ ] Create `Exit` node (cleanup/finalization)
- [ ] Define conditional edges based on task type
- [ ] Compile graph with `workflow.compile()`

### Streaming Adapter
- [ ] Create `kit/graph/streaming.py`
- [ ] Connect LangGraph `astream()` to Connect-RPC generator
- [ ] Handle chunk serialization for `GenerateStoryResponse`
- [ ] Test end-to-end streaming (Go → Python → Go)

### Verification
- [ ] "Hello World" workflow triggers via RPC
- [ ] Streaming content works end-to-end
- [ ] State initializes and updates correctly
- [ ] Graph compiles without errors

---

## Task 05: Agent - Story Generator (The "Writer")

**Goal:** Build creative writing agent with world context integration.

### LLM Setup
- [ ] Add `langchain-openai` or `langchain-anthropic` to dependencies
- [ ] Create `kit/agents/story_generator.py`
- [ ] Initialize `ChatOpenAI` or `ChatAnthropic` instance
- [ ] Create system prompt for creative writing
- [ ] Configure temperature, max_tokens, streaming options

### Context Injection
- [ ] Create `fetch_world_context(story_id, characters)` node
- [ ] Integrate Neo4j relationship lookup
- [ ] Create `query_semantic_memory(prompt)` node
- [ ] Integrate Qdrant search for past events
- [ ] Combine context into LLM prompt

### Write Node
- [ ] Implement `WriteContent` LangGraph node
- [ ] Stream output chunk-by-chunk
- [ ] Handle LLM errors/retries
- [ ] Add token counting for metrics

### Review Loop
- [ ] Create `self_review` node for tone/style check
- [ ] Define review criteria (genre compliance, length)
- [ ] Add revision loop if review fails

### Integration
- [ ] Add node to LangGraph workflow
- [ ] Connect to `GenerateStory` RPC handler
- [ ] Test with real LLM API key

### Verification
- [ ] Generator writes chapter based on prompt
- [ ] Neo4j relationships reflected in content
- [ ] Generated text matches genre constraints
- [ ] Streaming works without buffering issues

---

## Task 06: Agent - Story Analyst (The "Editor")

**Goal:** Build analyst agent for consistency, pacing, and tone review.

### Analyst Node Setup
- [ ] Create `kit/agents/analyst.py`
- [ ] Initialize LLM for analysis tasks
- [ ] Create system prompt for editorial review

### Consistency Check
- [ ] Implement `check_consistency(content, story_id)` node
- [ ] Query Neo4j for character relationships
- [ ] Query Qdrant for past plot points
- [ ] Compare content against existing world model
- [ ] Flag inconsistencies with explanations

### Sentiment Analysis
- [ ] Implement `calculate_sentiment(content)` node
- [ ] Option A: Use LLM for nuanced analysis
- [ ] Option B: Use `vaderSentiment` or `textblob`
- [ ] Return sentiment scores (positive, negative, neutral)

### Pacing Analysis
- [ ] Implement `analyze_pacing(content)` node
- [ ] Detect plot stagnation (low event density)
- [ ] Detect rapid shifts (abrupt tone changes)
- [ ] Return pacing score and recommendations

### Report Generator
- [ ] Format findings into `AnalyzeStoryResponse` proto
- [ ] Include all analysis sections
- [ ] Add actionable feedback for user
- [ ] Test with known good/bad content

### Verification
- [ ] Analyst identifies character inconsistencies
- [ ] Sentiment score reflects emotional tone
- [ ] Pacing analysis detects stagnation/rush
- [ ] Report is clear and actionable

---

## Task 07: Agent - Content Validator (The "Moderator")

**Goal:** Build safety and policy enforcement agent.

### Validator Node Setup
- [ ] Create `kit/agents/content_validator.py`
- [ ] Initialize LLM for safety evaluation
- [ ] Create "Safety System Prompt" template

### Rule-Based Checks
- [ ] Implement `rule_check(content)` node
- [ ] Create regex patterns for prohibited themes
- [ ] Create keyword blacklist
- [ ] Fast deterministic checks (sub-100ms)

### LLM Safety Check
- [ ] Implement `llm_safety_check(content)` node
- [ ] Evaluate nuanced policy violations
- [ ] Handle adversarial prompt attempts
- [ ] Return confidence score for decisions

### Violation Handling
- [ ] Implement `flag_violation(reason, category)` node
- [ ] Add flags to `StoryState.safety_flags`
- [ ] Categorize violations (hate, violence, explicit, etc.)
- [ ] Store violation history (optional)

### Configuration
- [ ] Create safety thresholds config
- [ ] Configure sensitivity per story genre
- [ ] Allow admin override settings

### Verification
- [ ] Validator flags prohibited content
- [ ] Rejection reasoning is clear
- [ ] Robust against adversarial prompts
- [ ] Integrates with `ValidateContent` RPC

---

## Task 08: Agent - Submission Approver (The "Gatekeeper")

**Goal:** Build final decision agent combining analyst + validator inputs.

### Approver Node Setup
- [ ] Create `kit/agents/submission_approver.py`
- [ ] Initialize decision logic module

### Reputation Check
- [ ] Implement `reputation_check(user_id)` node
- [ ] Fetch user's historical approval rate from Postgres
- [ ] Calculate trust score (0.0 - 1.0)
- [ ] Handle new users (default reputation)

### Content Score Aggregation
- [ ] Implement `content_score_threshold()` node
- [ ] Combine sentiment score from Analyst
- [ ] Combine consistency score from Analyst
- [ ] Combine safety flags from Validator
- [ ] Weight scores appropriately

### Decision Logic
- [ ] Implement `final_decision(approval_score)` node
- [ ] **Auto-Approve:** High reputation + no flags + high scores
- [ ] **Auto-Reject:** Critical safety flags OR very low scores
- [ ] **Manual Review:** Borderline cases
- [ ] Return decision with reasoning

### Integration
- [ ] Connect to `ValidateContent` RPC response
- [ ] Return `Approved`/`Rejected`/`ManualReview` status
- [ ] Include all metadata in response

### Verification
- [ ] Trusted users auto-approved for quality content
- [ ] Prohibited submissions auto-rejected
- [ ] Go backend receives correct status + metadata
- [ ] Edge cases handled (new users, missing data)

---

## Task 09: End-to-End Integration & Testing

**Goal:** Full system testing from Go backend to Python AI Toolkit.

### Integration Test Setup
- [ ] Create Go test script for RPC methods
- [ ] Mock LLM responses (avoid API costs)
- [ ] Test `GenerateStory` streaming
- [ ] Test `AnalyzeStory` with sample content
- [ ] Test `ValidateContent` flow
- [ ] Test `SubmissionApprove` flow

### E2E Workflow Tests
- [ ] Story Generation flow (start → finish)
- [ ] Story Analysis flow (good + bad content)
- [ ] Content Validation flow
- [ ] Submission Approval flow
- [ ] Error recovery scenarios

### Performance Analysis
- [ ] Measure LangGraph node execution times
- [ ] Test concurrent request handling
- [ ] Identify bottlenecks
- [ ] Document latency baselines

### Deployment Configuration
- [ ] Create `kit/Dockerfile`
- [ ] Update `docker-compose.yaml` with `kit` service
- [ ] Configure environment variables for production
- [ ] Add LLM API keys to secrets management
- [ ] Test container startup

### Verification
- [ ] All integration tests pass reliably
- [ ] Go-Python communication stable under load
- [ ] System ready for production deployment

---

## Task 10: Observability (Metrics, Logs, Tracing)

**Goal:** Integrate Python service into Fiente observability stack.

### OTEL Initialization
- [ ] Add `opentelemetry-api`, `opentelemetry-sdk` to dependencies
- [ ] Add `opentelemetry-exporter-otlp` to dependencies
- [ ] Create `kit/services/telemetry.py`
- [ ] Setup `TracerProvider` targeting `otel-collector:4318`
- [ ] Setup `MeterProvider` for metrics
- [ ] Configure resource attributes (service.name = "kit")

### Auto-Instrumentation
- [ ] Add `opentelemetry-instrumentation-fastapi`
- [ ] Add `opentelemetry-instrumentation-sqlalchemy`
- [ ] Add `opentelemetry-instrumentation-httpx`
- [ ] Apply instrumentors in server startup
- [ ] Verify traces appear in Jaeger

### Custom Instrumentation
- [ ] Create custom spans for each LangGraph node
- [ ] Track `llm_tokens_total` metric
- [ ] Track `agent_execution_seconds` metric
- [ ] Add span attributes for story_id, user_id

### Logging Strategy
- [ ] Configure `structlog` JSON output to stdout
- [ ] Add trace/span IDs to log context
- [ ] Configure promtail or otel-collector for log scraping
- [ ] Verify logs appear in Loki

### Grafana Dashboards
- [ ] Create Python service health dashboard
- [ ] Create AI performance dashboard
- [ ] Import token usage visualization
- [ ] Set up alerts for error rates

### Docker Compose Update
- [ ] Add `kit` service to `server/docker-compose.yaml`
- [ ] Configure all environment variables
- [ ] Add depends_on for db, otel-collector, neo4j, qdrant
- [ ] Test service discovery within Docker network

### Verification
- [ ] Traces from Python appear in Jaeger
- [ ] FastAPI metrics appear in Prometheus
- [ ] Structured logs appear in Loki
- [ ] Token usage visible in Grafana

---

## Cross-Task Dependencies

See `CROSS-TASK-DEPENDENCIES.md` for detailed dependency mapping.
