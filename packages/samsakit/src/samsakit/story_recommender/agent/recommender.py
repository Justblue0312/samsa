"""Recommendation Agent - Final streaming recommendation generation."""

import json
import logging
from collections.abc import AsyncGenerator
from typing import Any

from pydantic import BaseModel
from pydantic_ai import Agent

from samsakit.config import settings

from .context_signals import ContextPayload
from .prompts import SYSTEM_PROMPT, build_recommendation_prompt

logger = logging.getLogger(__name__)


class StoryRecommendation(BaseModel):
    """A single story recommendation."""

    story_id: str
    title: str
    synopsis: str = ""
    author: str = ""
    genres: list[str] = []
    reason: str
    signal_source: str
    confidence_note: str


async def generate_recommendations(
    user_id: str,
    prompt: str,
    stories: list[dict[str, Any]],
    context: ContextPayload,
    limit: int,
    token_tracker,
    stream,
) -> list[dict]:
    """
    Generate story recommendations using the LLM.

    Streams recommendations one-by-one to the client.
    """
    if not stories:
        logger.warning("No stories available for recommendations")
        return []

    # Prepare stories data for the prompt
    stories_data = json.dumps(stories[:20], indent=2)  # Limit to 20 for prompt size

    context_payload = context.to_prompt()

    # Build the prompt
    user_prompt = build_recommendation_prompt(
        context_payload=context_payload,
        stories_data=stories_data,
        user_prompt=prompt,
        limit=limit,
    )

    try:
        # Run the agent
        logger.info("Generating recommendations...")
        agent = Agent(
            model=settings.openai_model,
            output_type=list[StoryRecommendation],
        )
        result = await agent.run(user_prompt)

        recommendations = result.output

        # Stream each recommendation
        results = []
        for rec in recommendations[:limit]:
            rec_dict = rec.model_dump()
            if stream:
                await stream.send(
                    {
                        "event": "recommendation",
                        "data": rec_dict,
                    }
                )
            results.append(rec_dict)

        # Track usage
        if token_tracker and hasattr(result, 'usage') and result.usage:
            token_tracker.add_usage(
                agent_name="recommendation",
                usage=result.usage,
                elapsed_ms=getattr(result, 'elapsed_ms', 0),
            )

        return results

    except Exception as e:
        logger.exception("Recommendation generation failed")
        raise


class StreamingRecommendationAgent:
    """Streaming recommendation agent with tool calling support."""

    async def generate(
        self,
        user_id: str,
        prompt: str,
        stories: list[dict],
        context: ContextPayload,
        limit: int,
    ) -> AsyncGenerator[dict]:
        """Generate recommendations with streaming."""
        results = await generate_recommendations(
            user_id=user_id,
            prompt=prompt,
            stories=stories,
            context=context,
            limit=limit,
            token_tracker=None,
            stream=None,
        )

        for rec in results:
            yield rec
