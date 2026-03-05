# Task 02: Python Service Setup (FastAPI + Connect-RPC)

## Overview
This task involves creating the basic FastAPI server in the `kit` project that will host the AI Agents and handle incoming Connect-RPC requests from the Go backend.

## Objectives
*   Configure the FastAPI application with `uvicorn` and `connect-rpc-python`.
*   **[Observability]** Initialize OpenTelemetry (OTEL) SDK to export to `otel-collector`.
*   Implement a "ping" or "echo" handler in Python to verify the RPC connection.
*   Setup the `settings.py` to load configuration from environment variables.

## Components
*   **Server Setup (`kit/api/server.py`):**
    *   Initialize FastAPI.
    *   **[Observability]** Add `FastAPIInstrumentor` to trace all RPC requests.
    *   Mount the generated `AgentService` handler using Connect-RPC's Python middleware.
*   **Service Handlers (`kit/api/handlers.py`):**
    *   Initial `AgentService` class implementing `GenerateStory`, `AnalyzeStory`, and `ValidateContent`.
    *   **[Observability]** Use `structlog` for structured JSON logging compatible with Loki.
    *   Initial implementations should return mocked data to test the Go-Python connection.
*   **Logging:** Setup structured logging (e.g., `structlog`) for debugging and monitoring.

## Success Criteria
*   Python server starts via `uv run kit`.
*   A `curl` request to `POST /kit.v1.AgentService/AnalyzeStory` (Connect-RPC over HTTP) returns a successful response.
*   Go client can call the Python server successfully.

## Next Steps
*   Implement Database Connectors in Task 03.
