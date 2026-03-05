# AI Agent-Based Product Recommendation System

### Complete Architecture & Implementation Guide

---

## What This System Is

A product recommendation engine that replaces heavy ML pipelines (collaborative filtering, matrix factorisation, retraining jobs) with an **LLM-powered multi-agent system**. It gathers ranked user context signals, generates and validates SQL queries through a two-agent safety loop, then streams personalised recommendations to the client in real-time over WebSocket.

**Core philosophy:**

- No model training, no feature stores, no retraining cycles
- Every recommendation is explainable — the agent tells the user _why_
- The database is never touched by raw LLM output — only validated, approved SQL runs
- The client sees results arriving progressively, not after a long wait

---

## Overall Workflow

```
┌─────────────────────────────────────────────────────────────────────────┐
│  CLIENT                                                                  │
│  WebSocket connect → ws://api/ws/recommendations/{user_id}              │
│  Send: { "prompt": "...", "limit": 5 }                                  │
└───────────────────────────┬─────────────────────────────────────────────┘
                            │ WebSocket
                            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  FASTAPI WEBSOCKET HANDLER                                               │
│  1. Accept connection                                                    │
│  2. Send cached results immediately (if available) ──────────────────┐  │
│  3. Start background pipeline                                         │  │
└───────────────────────────┬──────────────────────────────────────────┼──┘
                            │                                           │
         ┌──────────────────▼──────────────────┐                       │
         │  CONTEXT BUILDER                     │                       │
         │  Gathers 6 signals in parallel       │                       │
         │  Scores by weight + freshness decay  │                       │
         │  Sorts: highest intent first         │                       │
         └──────────────────┬──────────────────┘                       │
                            │ Ranked ContextPayload                     │
         ┌──────────────────▼──────────────────┐                       │
         │  SQL AGENT PIPELINE                  │                       │
         │                                      │                       │
         │  [Agent 1] Query Generator           │                       │
         │      ↓ raw SQL                       │                       │
         │  [Free]  Deterministic Checker        │                       │
         │      ↓ pass / flag error             │                       │
         │  [Agent 2] Query Validator           │                       │
         │      ↓ approved / corrected SQL      │                       │
         │  [Gatekeeper] Sandboxed Executor     │                       │
         │      ↓ query results (read-only)     │                       │
         └──────────────────┬──────────────────┘                       │
                            │ DB results                                │
         ┌──────────────────▼──────────────────┐                       │
         │  RECOMMENDATION AGENT                │                       │
         │  LLM reasons over context + DB data  │                       │
         │  Streams recommendations 1-by-1      │                       │
         └──────────────────┬──────────────────┘                       │
                            │ stream of events                          │
         ┌──────────────────▼──────────────────┐                       │
         │  TOKEN TRACKER                       │                       │
         │  tiktoken pre-flight gate            │                       │
         │  pydantic-ai RunUsage accounting     │                       │
         │  Cost summary in "done" event        │                       │
         └──────────────────┬──────────────────┘                       │
                            │                                           │
┌───────────────────────────▼───────────────────────────────────────────▼──┐
│  CLIENT receives WebSocket event stream:                                  │
│                                                                           │
│  cached_results  → (instant, if cache hit)  ◄──────────────────────────┘ │
│  start           → "building_context"                                     │
│  context_ready   → signals used + weights                                 │
│  heartbeat       → every 2s while LLM thinks                             │
│  recommendation  → product 1  (streams as ready)                         │
│  recommendation  → product 2                                              │
│  recommendation  → product N                                              │
│  done            → total count + token usage + cost                       │
└───────────────────────────────────────────────────────────────────────────┘
```

---

## Project Structure

```
recommendation_system/
├── main.py                          # FastAPI app, WebSocket handler, lifespan
│
├── agent/
│   ├── context_signals.py           # Signal types, base weights, freshness decay rules
│   ├── schema_registry.py           # Allowed tables, columns, field descriptions
│   ├── sql_validator.py             # Deterministic rule-based SQL checker (free)
│   ├── query_generator_agent.py     # pydantic-ai Agent 1 — produces SQL
│   ├── query_validator_agent.py     # pydantic-ai Agent 2 — approves/corrects/rejects
│   ├── sql_orchestrator.py          # Wires Agent 1 → checker → Agent 2 → retry loop
│   ├── db_executor.py               # Read-only sandboxed DB executor (final gate)
│   ├── recommender.py               # Streaming recommendation agent (tool-calling)
│   └── prompts.py                   # Context signals → ranked prompt builder
│
├── services/
│   ├── context_builder.py           # Gathers all 6 signals, applies weights & decay
│   ├── product_service.py           # search_products_db, get_related_products_db
│   ├── user_service.py              # get_cart_items, get_recently_viewed, get_purchase_history
│   └── cache_service.py             # Redis stale-while-revalidate cache
│
├── monitoring/
│   ├── token_tracker.py             # tiktoken pre-flight + pydantic-ai RunUsage accounting
│   └── cost_comparison.py           # AI agent vs traditional ML cost report
│
├── models/
│   ├── schemas.py                   # Pydantic request/response models
│   └── database.py                  # asyncpg pool setup
│
├── requirements.txt
└── .env
```

---

## Part 1 — Context Signals & Ranking

Before any LLM is called, the system builds a rich picture of user intent from 6 signal sources. The higher the signal's weight, the more prominent it is in the LLM prompt.

### Signal Hierarchy

```
Signal            Base Weight   Freshness Decay    What It Represents
─────────────────────────────────────────────────────────────────────────
user_prompt         1.00        None (always fresh)  Explicit typed intent
cart_items          0.85        48 hours             Active purchase intent
recently_viewed     0.70        24 hours             Active browsing interest
same_category       0.50        72 hours             Category affinity
same_supplier       0.35        None                 Brand/supplier loyalty
purchase_history    0.25        None                 Background preference
─────────────────────────────────────────────────────────────────────────
```

Freshness decay reduces a signal's effective weight linearly as it ages. A cart item added 2 days ago has near-minimum influence; one added 10 minutes ago carries its full 0.85 weight.

```
effective_weight = base_weight × freshness_multiplier
freshness_multiplier = 1.0 → 0.3  (decays linearly over the decay window)
```

All signals are gathered in parallel using `asyncio.gather` (cart, viewed history, purchase history fetched simultaneously), then sorted descending by `effective_weight`. The top signals appear first in the prompt with explicit `⚡ HIGHEST PRIORITY` labels so the LLM knows what to focus on.

---

## Part 2 — SQL Agent Pipeline (Two-Agent Safety Loop)

The most critical safety layer. The LLM never touches the database directly — it only generates SQL through a controlled two-agent peer-review cycle with deterministic guardrails at every step.

### Three Layers of Defense

```
Layer 1 — FREE       DeterministicSQLChecker    Pure Python, regex rules, zero tokens
Layer 2 — LLM        Query Validator Agent      Reviews, corrects, or rejects
Layer 3 — FREE       Sandboxed DB Executor      read-only transaction, final check
```

### Schema Registry

The registry is the ground truth that both agents and the checker reference. It defines every table and column the LLM is _allowed_ to know about.

```
Table: products
  product_id   (uuid)        filterable, selectable
  name         (text)        filterable, selectable
  category     (text)        filterable, selectable
  supplier_id  (uuid)        filterable, selectable
  price        (numeric)     filterable, selectable
  stock        (integer)     filterable, selectable
  is_active    (boolean)     filterable, selectable
  description  (text)        selectable only — NOT filterable

Table: user_orders              ← ALWAYS requires WHERE user_id = ?
Table: product_views            ← ALWAYS requires WHERE user_id = ?
Table: suppliers

FORBIDDEN (agents never see these): users, payments, sessions, audit_logs
```

### Deterministic Checker (Layer 1 — no LLM cost)

Runs before the Validator Agent. Catches obvious violations instantly:

- Only `SELECT` statements — no `INSERT`, `UPDATE`, `DELETE`, `DROP`, etc.
- No forbidden SQL keywords (`--`, `/*`, `xp_`, `information_schema`)
- No forbidden tables in `FROM` or `JOIN` clauses
- All referenced tables must exist in the registry
- `user_orders` and `product_views` **must** have a `WHERE user_id = ?` filter
- `LIMIT` must not exceed the table's `max_rows_per_query`

### Agent 1 — Query Generator (pydantic-ai)

```
Input:  Ranked context payload + user_id
Output: GeneratedQuery { sql, intent, tables_used }
Model:  gpt-4o-mini  |  UsageLimits: 3 requests, 4000 tokens max
```

Knows only the schema registry and the rules. Produces one SELECT query targeting the most relevant data for this user's context.

### Agent 2 — Query Validator (pydantic-ai)

```
Input:  original_sql + deterministic error (if any) + user_id
Output: ValidationResult { approved, corrected_sql, rejection_reason, changes_made }
Model:  gpt-4o-mini  |  UsageLimits: 2 requests, 3000 tokens max
```

Can do one of three things:

- **Approve** — SQL is clean, pass to executor
- **Correct** — SQL has fixable issues, return corrected version (triggers retry)
- **Reject** — SQL cannot be fixed, raise error

Up to `MAX_CORRECTION_ATTEMPTS = 2` cycles before failing.

### Sandboxed Executor (Layer 3 — final gate)

```python
async with conn.transaction(readonly=True):
    # Runs one last deterministic check
    # Executes with parameterised binding only
    rows = await conn.fetch(safe_sql, user_id)
```

All SQL runs in a PostgreSQL `readonly=True` transaction. If the database user is configured with `GRANT SELECT ONLY`, a write operation would fail at the DB level even if it slipped through all prior checks.

---

## Part 3 — Recommendation Agent (Streaming)

After the DB results are ready, the Recommendation Agent generates final product recommendations using the LLM with tool-calling. The key latency features:

### Three Latency Tactics

**Tactic 1 — Stale-while-revalidate cache**
If the user has recommendations cached from a prior visit, send them immediately via the `cached_results` WebSocket event. The fresh generation continues in the background. The user sees something useful within milliseconds.

**Tactic 2 — Progressive streaming**
Each recommendation is pushed to the client as soon as the LLM produces it, rather than waiting for all N to complete. The user watches results appear one by one.

**Tactic 3 — Heartbeat keep-alive**
A concurrent coroutine pings the WebSocket every 2 seconds while the LLM is thinking. This prevents browser/mobile WebSocket timeouts during slow LLM calls.

### LLM Tool Contracts (Strict Whitelist)

The Recommendation Agent may only call 3 database tools, capped at 5 total tool calls per request:

```
search_products        keyword, category, supplier_id, max_price, limit (1–20)
get_product_details    product_id
get_related_products   product_id, relation_type (enum), limit (1–10)
```

Every output recommendation must include `signal_source` — one of:
`user_prompt | cart | viewed | category | supplier | history`

This makes every recommendation traceable back to the context signal that drove it.

---

## Part 4 — Token Tracking & Cost Visibility

### tiktoken vs. pydantic-ai RunUsage — Different Roles

```
tiktoken              → PRE-FLIGHT   Estimate tokens before the API call
                                     Gate if context is too large (avoids waste)

pydantic-ai RunUsage  → POST-CALL    Exact counts from model response headers
                                     Per-agent input/output/total/requests breakdown
```

Both feed into `TokenTracker`, which accumulates usage across all agents in a single request and reports a full cost summary in the `done` WebSocket event.

### Per-Agent Token Budget (enforced by pydantic-ai `UsageLimits`)

```
query_generator_agent    max 4000 tokens, max 3 LLM requests
query_validator_agent    max 3000 tokens, max 2 LLM requests
recommendation_agent     max 8000 tokens, max 5 tool calls
─────────────────────────────────────────────────────────────
Pre-flight gate          Raise early if context > 8000 tokens (tiktoken estimate)
```

### Cost Comparison Shape (AI Agent vs Traditional ML)

```
AI Agent per request:
  query_generator:    ~400 tokens  →  $0.000072
  query_validator:    ~550 tokens  →  $0.000099
  recommendation:    ~1500 tokens  →  $0.000270
  ─────────────────────────────────────────────
  Total per request: ~$0.000441   (gpt-4o-mini pricing)
  Monthly @ 100k req: ~$44

Traditional ML:
  Inference per request: ~$0.000002  (cheap — cached model)
  Retraining job/month:  ~$45–200    (EC2/Vertex AI)
  Feature engineering + MLOps overhead: significant
  ─────────────────────────────────────────────────
  Total monthly @ 100k: ~$50–205 + team time

Verdict:
  At moderate volume, cost is comparable.
  AI agent eliminates the retraining infrastructure and MLOps overhead.
  Break-even shifts in favour of AI agent when factoring in engineer time.
```

---

## Part 5 — WebSocket Event Protocol

Full client/server contract:

```
CLIENT → SERVER (once, on connect)
───────────────────────────────────────────────────────
{
  "prompt": "I need a gift for a gamer under $50",
  "limit": 5
}

SERVER → CLIENT (sequence of push events)
───────────────────────────────────────────────────────
{ "event": "cached_results",
  "data": { "recommendations": [...], "note": "Refreshing..." }}  ← instant if cache hit

{ "event": "start",
  "data": { "status": "building_context" }}

{ "event": "context_ready",
  "data": { "signals_used": [
      { "type": "user_prompt",    "weight": 1.0,  "summary": "User wants: gaming gift <$50" },
      { "type": "cart_items",     "weight": 0.82, "summary": "Cart: 2 items" },
      { "type": "recently_viewed","weight": 0.65, "summary": "Viewed 7 products" },
      ...
  ]}}

{ "event": "heartbeat",      "data": { "status": "processing" }}   ← every 2s

{ "event": "recommendation",
  "data": {
    "product_id": "abc-123",
    "name": "SteelSeries Arctis Nova 3",
    "price": 47.99,
    "reason": "Highly rated gaming headset within budget, matching your recent headphone browsing",
    "signal_source": "recently_viewed",
    "confidence_note": "high"
  }}

{ "event": "recommendation", "data": { ... }}   ← streams 1-by-1

{ "event": "done",
  "data": {
    "total": 5,
    "token_usage": {
      "agents": [
        { "name": "query_generator",  "total_tokens": 412,  "elapsed_ms": 780,  "cost_usd": 0.000074 },
        { "name": "query_validator",  "total_tokens": 534,  "elapsed_ms": 620,  "cost_usd": 0.000096 },
        { "name": "recommendation",   "total_tokens": 1480, "elapsed_ms": 2100, "cost_usd": 0.000266 }
      ],
      "totals": {
        "total_tokens": 2426,
        "cost_usd": 0.000436,
        "elapsed_ms": 3500
      }
    }
  }}

{ "event": "error", "data": { "message": "..." }}   ← only on failure
───────────────────────────────────────────────────────
```

---

## Part 6 — All Guardrails in One Place

```
Guardrail                         Where Applied               Type
═══════════════════════════════════════════════════════════════════════════════
Schema registry whitelist         sql_orchestrator            Structural
Only SELECT allowed               DeterministicSQLChecker     Deterministic (free)
No forbidden tables               DeterministicSQLChecker     Deterministic (free)
user_id required on sensitive tbl DeterministicSQLChecker     Deterministic (free)
LIMIT capped per table            DeterministicSQLChecker     Deterministic (free)
No SQL injection keywords         DeterministicSQLChecker     Deterministic (free)
LLM validation + correction       query_validator_agent       LLM (pydantic-ai)
read-only DB transaction          db_executor                 Database
Tool call whitelist (3 tools)     tool_executor               Structural
Max 5 tool calls per request      tool_executor               Budget
Argument key sanitization         tool_executor               Structural
No hallucinated product IDs       Recommendation agent rules  Prompt
No already-purchased repeats      post-generation filter      Code
tiktoken pre-flight gate          token_tracker               Budget
Per-agent UsageLimits             pydantic-ai UsageLimits     Budget
Hard 15s asyncio.timeout          WebSocket handler           Timeout
Prompt injection: user input      context_builder             Prompt design
  treated as data, not instruction
═══════════════════════════════════════════════════════════════════════════════
```

---

## Requirements

```txt
fastapi
uvicorn[standard]
redis[hiredis]
pydantic-ai[openai]        # pydantic-ai with OpenAI provider
pydantic>=2.0
asyncpg                    # async PostgreSQL
tiktoken                   # pre-flight token estimation
python-dotenv
logfire                    # optional: pydantic-ai native observability dashboard
```

---

## Key Design Decisions — Quick Reference

| Decision         | Choice                                        | Reason                                                        |
| ---------------- | --------------------------------------------- | ------------------------------------------------------------- |
| Delivery         | WebSocket with streaming                      | Client sees results progressively, no long blank wait         |
| Context          | 6 signals, weighted + freshness decay         | Prioritises explicit intent over stale history                |
| SQL safety       | Deterministic check first, then LLM Validator | Cheap layer catches ~80% of issues before burning tokens      |
| DB interaction   | Registered tools only, read-only transaction  | LLM can never mutate data or escape the schema                |
| Agent framework  | pydantic-ai                                   | Typed outputs, built-in UsageLimits, RunUsage, tool support   |
| Token accounting | tiktoken (pre) + RunUsage (post)              | Pre-flight trims waste; post-call tracks exact cost per agent |
| Latency          | Cache → heartbeat → stream                    | Three-layer UX: instant + keep-alive + progressive delivery   |
| No ML pipeline   | LLM reasoning replaces training               | Explainable, context-aware, zero retraining overhead          |
