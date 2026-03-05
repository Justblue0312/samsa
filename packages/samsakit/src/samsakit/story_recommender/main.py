"""Main application with Connect-RPC and WebSocket mounts."""

import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware

from samsakit.config import settings
from samsakit.database import close_db, init_db
from samsakit.proto.samsa.api.recommendation.recommendation_service_connect import (  # type: ignore[import-untyped]
    RecommendationServiceASGIApplication,
)

from .connect_service import RecommendationServiceConnect

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan handler."""
    logger.info("Starting Story Recommender service...")
    await init_db()
    logger.info("Database initialized")
    yield
    logger.info("Shutting down Story Recommender service...")
    await close_db()


# Create FastAPI app
app = FastAPI(
    title="Story Recommender",
    description="AI-powered story recommendation service with Connect-RPC support",
    version="0.1.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Create Connect-RPC service
connect_service = RecommendationServiceConnect()
connect_app = RecommendationServiceASGIApplication(
    service=connect_service,
)

# Mount Connect-RPC application
app.mount("/samsa.api.recommendation.v1.RecommendationService", connect_app)


# Health check endpoint (REST)
@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "story-recommender"}


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "service": "Story Recommender",
        "version": "0.1.0",
        "connect_rpc": "/samsa.api.recommendation.v1.RecommendationService",
        "websocket": "/ws/recommendations/{user_id}",
    }


# WebSocket endpoint for streaming recommendations
@app.websocket("/ws/recommendations/{user_id}")
async def websocket_recommendations(websocket: WebSocket, user_id: str):
    """WebSocket endpoint for real-time streaming recommendations."""
    await websocket.accept()

    try:
        # Receive initial request
        data = await websocket.receive_json()
        prompt = data.get("prompt", "")
        limit = data.get("limit", 5)

        # Create a simple stream interface for WebSocket
        class WebSocketStream:
            async def send(self, message: dict):
                await websocket.send_json(message)

        stream = WebSocketStream()

        # Initialize services
        from samsakit.monitoring import TokenTracker

        from .agent.db_executor import close_db_executor, create_db_executor
        from .agent.recommender import generate_recommendations
        from .services.cache_service import CacheService
        from .services.context_builder import ContextBuilder
        from .services.story_service import StoryService

        db_executor = await create_db_executor(settings.database_url)
        story_service = StoryService(db_executor)
        cache_service = CacheService()
        await cache_service.set_redis_from_url(settings.redis_url)

        context_builder = ContextBuilder(story_service, cache_service)
        token_tracker = TokenTracker()

        try:
            # Send start
            await stream.send({"event": "start", "data": {"status": "building_context"}})

            # Check cache
            cached = await cache_service.get_cached_recommendations(user_id, prompt)
            if cached:
                await stream.send(
                    {
                        "event": "cached_results",
                        "data": {"recommendations": cached, "note": "Refreshing..."},
                    }
                )

            # Build context
            context = await context_builder.build_context(user_id, prompt)
            await stream.send({"event": "context_ready", "data": {"signals_used": context.signals}})

            # Heartbeat
            heartbeat_task = asyncio.create_task(send_heartbeat(websocket))

            # Get stories
            stories = await story_service.get_stories_for_recommendations(user_id, context)

            # Generate recommendations
            recommendations = await generate_recommendations(
                user_id=user_id,
                prompt=prompt,
                stories=stories,
                context=context,
                limit=limit,
                token_tracker=token_tracker,
                stream=stream,
            )

            # Cancel heartbeat
            heartbeat_task.cancel()

            # Cache
            await cache_service.cache_recommendations(user_id, prompt, [r["story_id"] for r in recommendations])

            # Send done
            usage = token_tracker.get_summary()
            await stream.send(
                {
                    "event": "done",
                    "data": {"total": len(recommendations), "token_usage": usage},
                }
            )

        finally:
            await close_db_executor(db_executor)
            await cache_service.close()

    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected: {user_id}")
    except Exception as e:
        logger.exception("WebSocket error")
        await websocket.send_json({"event": "error", "data": {"message": str(e)}})
    finally:
        await websocket.close()


async def send_heartbeat(websocket: WebSocket):
    """Send heartbeat every 2 seconds to keep connection alive."""
    while True:
        try:
            await asyncio.sleep(2)
            await websocket.send_json({"event": "heartbeat", "data": {"status": "processing"}})
        except asyncio.CancelledError:
            break
        except Exception:
            break


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "samsakit.story_recommender.main:app",
        host=settings.ws_host,
        port=settings.ws_port,
        reload=True,
    )
