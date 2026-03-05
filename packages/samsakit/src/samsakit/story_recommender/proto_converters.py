"""Convert between internal models and protobuf messages."""

from typing import TYPE_CHECKING

from samsakit.proto.samsa.api.recommendation import (
    recommendation_service_pb2 as pb2,
)

if TYPE_CHECKING:
    from .agent.context_signals import ContextPayload


def story_to_proto(story: dict) -> pb2.StoryRecommendation:
    """Convert story dict to proto."""
    return pb2.StoryRecommendation(
        story_id=story.get("story_id", ""),
        title=story.get("title", ""),
        synopsis=story.get("synopsis", ""),
        author=story.get("author", ""),
        genres=story.get("genres", []),
        reason=story.get("reason", ""),
        signal_source=story.get("signal_source", ""),
        confidence_note=story.get("confidence_note", "medium"),
    )


def context_to_proto(context: "ContextPayload") -> pb2.ContextSignals:
    """Convert context to proto."""
    signals = [
        pb2.ContextSignals.Signal(
            type=signal.type.value,  # type: ignore[attr-defined]
            weight=signal.effective_weight,  # type: ignore[attr-defined]
            summary=str(signal.data),  # type: ignore[attr-defined]
        )
        for signal in context.signals  # type: ignore[attr-defined]
    ]
    return pb2.ContextSignals(signals=signals)


def cached_results_to_proto(
    cached: list[dict], note: str
) -> pb2.CachedResults:
    """Convert cached results to proto."""
    return pb2.CachedResults(
        stories=[story_to_proto(story) for story in cached],
        note=note,
    )
