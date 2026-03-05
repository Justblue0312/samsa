# Example 01: FastAPI + Connect-RPC

This example shows how to mount a **Connect-RPC** service onto a **FastAPI** application, allowing you to use FastAPI's middleware and routing alongside high-performance RPC.

### 1. Protobuf Definition (`story.proto`)
```protobuf
syntax = "proto3";
package kit.v1;

message GenerateRequest { string prompt = 1; }
message GenerateResponse { string chunk = 1; }

service StoryService {
  rpc Generate(GenerateRequest) returns (stream GenerateResponse);
}
```

### 2. Implementation (`api/server.py`)
```python
import asyncio
from fastapi import FastAPI
from connectrpc import ConnectRPC
from kit.gen.v1 import story_pb2, story_connect

app = FastAPI()

class StoryServiceHandler(story_connect.StoryServiceHandler):
    async def Generate(self, request, context):
        """Streaming RPC example"""
        words = ["Once", "upon", "a", "time...", "in", "a", "galaxy", "far", "away."]
        for word in words:
            yield story_pb2.GenerateResponse(chunk=word + " ")
            await asyncio.sleep(0.5)

# Initialize Connect-RPC
rpc = ConnectRPC()
rpc.add_service(StoryServiceHandler())

# Mount RPC handler into FastAPI
app.mount("/kit.v1.StoryService", rpc)

@app.get("/health")
async def health():
    return {"status": "ok"}
```

### Why use both?
*   **FastAPI:** Handles HTTP/1.1, Swagger docs, OAuth2, and simple JSON endpoints.
*   **Connect-RPC:** Handles high-performance HTTP/2 binary streaming, which is much more efficient for real-time AI generation than standard REST.
