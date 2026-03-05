"""Context signals for story recommendations."""

from dataclasses import dataclass, field
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any

from samsakit.config import settings


class SignalType(StrEnum):
    """Types of context signals."""

    USER_PROMPT = "user_prompt"
    BOOKMARKS = "bookmarks"
    FAVORITES = "favorites"
    VOTES = "votes"
    RECENTLY_VIEWED = "recently_viewed"
    SAME_GENRE = "same_genre"


@dataclass
class Signal:
    """A context signal with weight and metadata."""

    type: SignalType
    base_weight: float
    data: Any
    fetched_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    decay_window_hours: int = 24

    @property
    def effective_weight(self) -> float:
        """Calculate weight with freshness decay."""
        if self.type == SignalType.USER_PROMPT:
            return self.base_weight  # No decay for explicit intent

        age_hours = (datetime.now(UTC) - self.fetched_at).total_seconds() / 3600
        decay_factor = max(0.3, 1.0 - (age_hours / self.decay_window_hours))
        return self.base_weight * decay_factor

    def to_dict(self) -> dict:
        """Convert to dictionary for prompts."""
        return {
            "type": self.type.value,
            "weight": round(self.effective_weight, 2),
            "data": self.data,
        }


# Signal configurations
SIGNAL_CONFIGS: dict[SignalType, dict] = {
    SignalType.USER_PROMPT: {
        "base_weight": 1.00,
        "decay_window_hours": 0,  # No decay
    },
    SignalType.BOOKMARKS: {
        "base_weight": 0.85,
        "decay_window_hours": settings.decay_window_bookmarks,
    },
    SignalType.FAVORITES: {
        "base_weight": 0.80,
        "decay_window_hours": settings.decay_window_favorites,
    },
    SignalType.VOTES: {
        "base_weight": 0.75,
        "decay_window_hours": 0,  # Votes don't decay
    },
    SignalType.RECENTLY_VIEWED: {
        "base_weight": 0.70,
        "decay_window_hours": settings.decay_window_recently_viewed,
    },
    SignalType.SAME_GENRE: {
        "base_weight": 0.50,
        "decay_window_hours": settings.decay_window_same_genre,
    },
}


@dataclass
class ContextPayload:
    """Ranked context payload for LLM prompts."""

    user_id: str
    prompt: str
    signals: list[dict] = field(default_factory=list)
    top_signals: list[dict] = field(default_factory=list)

    def to_prompt(self) -> str:
        """Generate prompt text from context."""
        lines = [
            "## User Context",
            f"User ID: {self.user_id}",
            f"User Request: {self.prompt}",
            "",
            "## Context Signals (ranked by relevance)",
        ]

        for i, signal in enumerate(self.top_signals[:3], 1):
            priority = "⚡ HIGHEST PRIORITY" if i == 1 else ""
            lines.append(f"{i}. [{signal['type']}] weight={signal['weight']:.2f} {priority}")
            lines.append(f"   Data: {signal.get('summary', str(signal.get('data', '')))}")

        lines.append("")
        lines.append("## Available Stories")
        return "\n".join(lines)


def create_signal(
    signal_type: SignalType,
    data: Any,
    fetched_at: datetime | None = None,
) -> Signal:
    """Create a signal with proper configuration."""
    config = SIGNAL_CONFIGS[signal_type]
    return Signal(
        type=signal_type,
        base_weight=config["base_weight"],
        decay_window_hours=config["decay_window_hours"],
        data=data,
        fetched_at=fetched_at or datetime.now(UTC),
    )


def rank_signals(signals: list[Signal]) -> list[Signal]:
    """Rank signals by effective weight (highest first)."""
    return sorted(signals, key=lambda s: s.effective_weight, reverse=True)
