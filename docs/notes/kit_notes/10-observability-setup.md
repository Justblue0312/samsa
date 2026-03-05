# Task 10: Observability (Metrics, Logs, Tracing)

## Overview
This task involves integrating the `kit` Python service into the existing Fiente observability stack (OTEL Collector, Prometheus, Loki, Jaeger, Grafana).

## Objectives
*   Configure **OpenTelemetry (OTEL)** to export traces and metrics to the `otel-collector`.
*   Implement **Structured Logging** (JSON) compatible with Grafana Loki.
*   Track **AI-specific metrics** (Token usage, LLM latency, Agent success rates).
*   Add the `kit` service to the main `docker-compose.yaml`.

## Components
*   **OTEL Initialization (`kit/services/telemetry.py`):**
    *   Setup `TracerProvider` and `MeterProvider` targeting `http://otel-collector:4318`.
    *   Instrument FastAPI, SQLAlchemy, and HTTPX.
*   **Logging Strategy:**
    *   Use `structlog` to output JSON logs to `stdout`.
    *   Configure `promtail` or `otel-collector` to scrape these logs for Loki.
*   **AI Monitoring (LangGraph + PydanticAI):**
    *   Custom spans for each LangGraph node.
    *   Track `llm_tokens_total` and `agent_execution_seconds` metrics.
*   **Grafana Dashboards:**
    *   Create/import dashboards for Python service health and AI performance.

## Docker Compose Update
Add the `kit` service to `server/docker-compose.yaml`:
```yaml
  kit:
    build:
      context: ../kit
      dockerfile: Dockerfile
    container_name: fiente-kit
    environment:
      - DATABASE_URL=postgresql+asyncpg://${FIENTE_POSTGRES_USER}:${FIENTE_POSTGRES_PWD}@db:5432/${FIENTE_POSTGRES_DATABASE}
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
      - NEO4J_URI=bolt://neo4j:7687
      - QDRANT_HOST=qdrant
    depends_on:
      - db
      - otel-collector
```

## Success Criteria
*   Traces from Python service appear in Jaeger.
*   FastAPI metrics appear in Prometheus.
*   Structured logs appear in Loki.
*   Token usage is visible in Grafana.

## Next Steps
*   Incorporate OTEL into Task 02 and Task 04.
