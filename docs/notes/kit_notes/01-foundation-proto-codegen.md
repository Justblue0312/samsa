# Task 01: Proto Definition & Code Generation

## Overview
This task involves defining the communication contract between the **Go Server** and the **Python AI Toolkit** using Protocol Buffers and generating the necessary stubs for both languages.

## Objectives
*   Create a unified `.proto` definition for the `AgentService`.
*   Configure `buf` to generate Go and Python code.
*   Verify that Go can import the generated client and Python can import the generated server.

## Components
*   **Protocol Definition:** Create `kit/v1/service.proto` with the following:
    *   `StoryContext`: Shared metadata for all requests.
    *   `GenerateStory`: Server-side streaming RPC for real-time text generation.
    *   `AnalyzeStory`: Unary RPC for story insights.
    *   `ValidateContent`: Unary RPC for safety checks.
*   **Buf Configuration:**
    *   `buf.yaml`: Dependency management (e.g., Google APIs).
    *   `buf.gen.yaml`: Plugins for `go`, `connect-go`, `python`, and `grpc-python`.

## Success Criteria
*   `buf generate` runs without errors in the project root.
*   Python stubs appear in `kit/src/kit/gen/`.
*   Go stubs appear in `server/internal/gen/`.
*   A basic `main.go` can import the `AgentServiceClient` without compile errors.

## Next Steps
*   Setup the basic Python FastAPI server in Task 02.
