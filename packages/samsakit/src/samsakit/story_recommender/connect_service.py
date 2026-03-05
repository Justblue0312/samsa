"""Connect-RPC Recommendation Service implementation."""

import logging
from typing import AsyncIterator

from connectrpc.code import Code
from connectrpc.errors import ConnectError
from connectrpc.request import RequestContext

from samsakit.monitoring import TokenTracker
from samsakit.proto.samsa.api.recommendation import (
    recommendation_service_pb2 as pb2,
)
from samsakit.proto.samsa.api.recommendation.recommendation_service_connect import (
    RecommendationService,
)

from .agent.db_executor import create_db_executor, close_db_executor
from .agent.recommender import generate_recommendations
from samsakit.config import settings
from .proto_converters import (
    cached_results_to_proto,
    context_to_proto,
    story_to_proto,
)
from .services.cache_service import CacheService
from .services.context_builder import ContextBuilder
from .services.story_service import StoryService

logger = logging.getLogger(__name__)


class ProtoStreamCollector:
    """Collect proto messages for streaming response."""

    def __init__(self):
        self.messages: list[pb2.RecommendationResponse] = []

    async def send(self, message: dict) -> None:
        """Convert dict message to proto and collect."""
        event = message.get("event", "")
        data = message.get("data", {})

        if event == "recommendation":
            proto_msg = pb2.RecommendationResponse(
                event=event,
                recommendation=story_to_proto(data),
            )
            self.messages.append(proto_msg)
        elif event == "done":
            proto_msg = pb2.RecommendationResponse(
                event=event,
                total=data.get("total", 0),
            )
            self.messages.append(proto_msg)


class RecommendationServiceConnect(RecommendationService):
    """Connect-RPC implementation of RecommendationService."""

    async def get_recommendations(
        self,
        request: pb2.RecommendationRequest,
        ctx: RequestContext,
    ) -> AsyncIterator[pb2.RecommendationResponse]:
        """Stream story recommendations via Connect-RPC."""
        # Initialize services
        db_executor = await create_db_executor(settings.database_url)
        story_service = StoryService(db_executor)
        cache_service = CacheService()
        await cache_service.set_redis_from_url(settings.redis_url)
        context_builder = ContextBuilder(story_service, cache_service)
        token_tracker = TokenTracker()

        try:
            # Check cache first
            cached = await cache_service.get_cached_recommendations(
                request.user_id, request.prompt
            )
            if cached:
                yield pb2.RecommendationResponse(
                    event="cached_results",
                    cached_results=cached_results_to_proto(
                        cached, "Refreshing with latest data..."
                    ),
                )

            # Build context
            context = await context_builder.build_context(
                request.user_id, request.prompt
            )
            yield pb2.RecommendationResponse(
                event="context_ready",
                signals=context_to_proto(context),
            )

            # Get stories
            stories = await story_service.get_stories_for_recommendations(
                request.user_id, context
            )

            # Collect recommendations
            collector = ProtoStreamCollector()

            # Generate recommendations
            recommendations = await generate_recommendations(
                user_id=request.user_id,
                prompt=request.prompt,
                stories=stories,
                context=context,
                limit=request.limit,
                token_tracker=token_tracker,
                stream=collector,
            )

            # Stream each recommendation
            for rec in recommendations:
                yield pb2.RecommendationResponse(
                    event="recommendation",
                    recommendation=story_to_proto(rec),
                )

            # Send done with total
            yield pb2.RecommendationResponse(
                event="done",
                total=len(recommendations),
            )

        except Exception as e:
            logger.exception("Recommendation generation failed")
            yield pb2.RecommendationResponse(
                event="error",
                error=pb2.ErrorMessage(message=str(e)),
            )
        finally:
            await close_db_executor(db_executor)
            await cache_service.close()

    async def health(
        self,
        request: pb2.HealthCheckRequest,
        ctx: RequestContext,
    ) -> pb2.HealthCheckResponse:
        """Health check."""
        return pb2.HealthCheckResponse(
            status=pb2.HealthCheckResponse.ServingStatus.SERVING
        )
