# Kit Cross-Task Dependencies

This document maps dependencies between tasks to identify coordination needs and parallelization opportunities.

---

## Dependency Graph

```
Task 01 (Proto) ─────────────────┬────────────────> Task 02 (FastAPI)
                                 │                      │
                                 │                      ▼
                                 │              Task 03 (Databases)
                                 │                      │
                                 │                      ▼
                                 │              Task 04 (LangGraph)
                                 │                      │
                                 ▼                      │
                    Task 05 (Writer) <──────────────────┘
                                 │
                                 ▼
                    Task 06 (Analyst)
                                 │
                                 ▼
                    Task 07 (Validator)
                                 │
                                 ▼
                    Task 08 (Approver)
                                 │
                                 ▼
                    Task 09 (Integration)
                                 │
                                 ▼
                    Task 10 (Observability)
```

---

## Detailed Dependency Matrix

### Task 01: Proto Definition & Code Generation

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Blocks** | Task 02, Task 05-08 | Generated Python stubs required for all RPC handlers |
| **Blocks** | Server (Go) | Go stubs required for client implementation |
| **Required By** | None | This is the foundation task |

**Coordination Notes:**
- Must complete before any RPC handler implementation
- Proto changes after Task 02+ will require regenerating stubs
- Consider versioning strategy for proto definitions

---

### Task 02: Python Service Setup

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 01 | Needs generated proto stubs |
| **Blocks** | Task 04, Task 05-08 | All agents need RPC handler integration point |
| **Parallel With** | Task 03 | Database connectors can be built independently |

**Coordination Notes:**
- Server skeleton can start with placeholder handlers
- Connect-RPC mounting is critical path for all agents
- Logging setup affects all downstream tasks

---

### Task 03: Database Connectors

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 02 (partial) | Needs settings.py for DB config |
| **Blocks** | Task 05, Task 06, Task 08 | Agents need data access |
| **Parallel With** | Task 04 | LangGraph workflow can use mock data initially |

**Coordination Notes:**
- Postgres connector needed by Task 08 (reputation check)
- Neo4j connector needed by Task 05 (world context) and Task 06 (consistency)
- Qdrant connector needed by Task 05 (semantic memory) and Task 06 (plot points)
- Unit tests should validate connection pooling before agent integration

---

### Task 04: LangGraph Foundation

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 01, Task 02, Task 03 | Needs proto stubs, server, and data access |
| **Blocks** | Task 05, Task 06, Task 07, Task 08 | All agents are LangGraph nodes |
| **Parallel With** | None | Critical path item |

**Coordination Notes:**
- State definition must accommodate all agent outputs
- Streaming adapter is shared infrastructure for Task 05
- Base workflow routing affects how agents are chained
- Consider checkpoint persistence for long-running stories

---

### Task 05: Story Generator Agent

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 03, Task 04 | Needs DB connectors and LangGraph workflow |
| **Blocks** | Task 06, Task 07 | Analyst and Validator need generated content |
| **Parallel With** | None | First agent in pipeline |

**Coordination Notes:**
- LLM API key configuration required
- Token counting integration needed for Task 10 metrics
- Streaming output must match Connect-RPC generator interface
- Self-review loop may overlap with Task 06 functionality

---

### Task 06: Story Analyst Agent

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 05 (output format) | Needs to understand generated content structure |
| **Blocks** | Task 08 | Approver needs analyst scores |
| **Parallel With** | Task 07 | Validator can be built independently |

**Coordination Notes:**
- Consistency check overlaps with Task 05's world context lookup
- Sentiment analysis is standalone (can start early)
- Report format must match proto definition from Task 01
- May need to iterate on Task 05 if context injection is insufficient

---

### Task 07: Content Validator Agent

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 01 | Needs proto for response format |
| **Blocks** | Task 08 | Approver needs safety flags |
| **Parallel With** | Task 06 | Independent of Analyst logic |

**Coordination Notes:**
- Rule-based checks can be built without LLM
- LLM safety check needs separate API key consideration
- Safety thresholds may need tuning based on Task 08 feedback
- Adversarial prompt testing should involve security review

---

### Task 08: Submission Approver Agent

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 06, Task 07 | Needs analyst scores and validator flags |
| **Blocks** | Task 09 | Integration tests need full pipeline |
| **Parallel With** | None | Final agent in pipeline |

**Coordination Notes:**
- Reputation system requires Postgres schema (Task 03)
- Decision thresholds need product input (what's "high reputation"?)
- Manual review routing needs external system integration (future work)
- Edge cases: new users, missing analyst data, validator errors

---

### Task 09: End-to-End Integration & Testing

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 05-08 | All agents must be implemented |
| **Blocks** | Production deployment | Cannot deploy without testing |
| **Parallel With** | Task 10 (partial) | Observability can be tested alongside |

**Coordination Notes:**
- Mock LLM responses to avoid API costs during testing
- Performance baselines needed before production
- Docker Compose updates affect entire stack
- Test data seeding required for reproducible tests

---

### Task 10: Observability

| Dependency Type | Tasks Affected | Details |
|-----------------|----------------|---------|
| **Requires** | Task 02 (server), Task 04 (LangGraph) | Needs instrumentation points |
| **Blocks** | Production monitoring | Cannot monitor without setup |
| **Parallel With** | Task 09 | Can test observability during integration |

**Coordination Notes:**
- OTEL initialization should ideally be in Task 02 (noted as gap)
- Custom spans for LangGraph nodes need Task 04 cooperation
- Token metrics need integration with Task 05 LLM calls
- Grafana dashboards require all metrics to be flowing

---

## Critical Path Analysis

### Longest Path (Blocking Chain)
```
Task 01 → Task 02 → Task 04 → Task 05 → Task 06 → Task 08 → Task 09
```

**Total blocking tasks:** 7 of 10

### Parallelization Opportunities

| Parallel Group | Tasks | Notes |
|----------------|-------|-------|
| **Group A** | Task 03 | Can run alongside Task 02 |
| **Group B** | Task 07 | Can run alongside Task 06 |
| **Group C** | Task 10 (partial) | OTEL setup can start after Task 02 |

---

## Identified Gaps & Recommendations

### Gap 1: Observability Overlap
**Issue:** Task 02 and Task 10 both mention OTEL initialization.

**Recommendation:**
- Move OTEL SDK setup entirely to Task 10
- Task 02 should only add FastAPI instrumentation (as a consumer)
- Update Task 04 to reference Task 10 for span creation patterns

---

### Gap 2: Reputation System Schema
**Issue:** Task 08 needs user reputation from Postgres, but Task 03 doesn't define the schema.

**Recommendation:**
- Add to Task 03 checklist: "Create `user_reputation` table schema"
- Fields: `user_id`, `total_submissions`, `approved_count`, `rejected_count`, `trust_score`
- Add migration script for schema deployment

---

### Gap 3: LLM Configuration
**Issue:** Multiple tasks need LLM access but no central configuration task.

**Recommendation:**
- Add to Task 02 or Task 10: "Centralize LLM configuration"
- Create `kit/services/llm.py` with:
  - Provider selection (OpenAI, Anthropic, OpenRouter)
  - API key management
  - Rate limiting configuration
  - Fallback provider support

---

### Gap 4: Error Handling Strategy
**Issue:** No task covers cross-cutting error handling.

**Recommendation:**
- Add to Task 04: "Define error handling patterns for LangGraph nodes"
- Document: retry policies, circuit breakers, graceful degradation
- Ensure all agents follow consistent error response format

---

### Gap 5: Background Tasks (Dramatiq)
**Issue:** Example 05 exists but no task in main plan.

**Recommendation:**
- **Option A:** Add Task 11 for Dramatiq integration if long-running stories are needed
- **Option B:** Remove example if all generation will be streaming (real-time)
- Decision depends on expected story length and timeout requirements

---

### Gap 6: Security Middleware
**Issue:** No auth/authz between Go ↔ Python.

**Recommendation:**
- Add to Task 02: "Add API key validation middleware"
- Go server signs requests with shared secret
- Python validates signature before processing
- Prevents unauthorized RPC calls

---

## Task Ordering Recommendation

### Phase 1: Foundation (Week 1-2)
1. Task 01 - Proto Definition
2. Task 02 - Python Service Setup
3. Task 03 - Database Connectors (parallel with Task 02)

### Phase 2: Core Workflow (Week 3-4)
4. Task 04 - LangGraph Foundation
5. Task 05 - Story Generator
6. Task 10 (partial) - OTEL initialization (parallel with Task 05)

### Phase 3: Agent Pipeline (Week 5-6)
7. Task 06 - Story Analyst
8. Task 07 - Content Validator (parallel with Task 06)
9. Task 08 - Submission Approver

### Phase 4: Production Ready (Week 7-8)
10. Task 09 - Integration & Testing
11. Task 10 (complete) - Full observability
12. Task 11 (optional) - Dramatiq background tasks

---

## Shared Resources

| Resource | Used By | Coordination Needed |
|----------|---------|---------------------|
| `StoryState` | Task 04, 05, 06, 07, 08 | Task 04 owns; others extend |
| `kit/settings.py` | All tasks | Central config; avoid conflicts |
| LLM API Keys | Task 05, 06, 07 | Share via Task 10 config |
| Proto Stubs | Task 02, 05, 06, 07, 08 | Task 01 owns; regenerate on changes |
| Docker Compose | Task 09, 10 | Coordinate service definitions |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Proto changes break handlers | Medium | High | Version proto files; test regeneration early |
| LLM API costs during testing | High | Medium | Mock responses in Task 09; use cheap models for dev |
| Database connection exhaustion | Medium | High | Test connection pooling in Task 03; add limits |
| LangGraph streaming complexity | Medium | High | Prototype streaming adapter in Task 04 early |
| Observability gaps in production | Low | High | Test dashboards in Task 10 before Task 09 completes |
