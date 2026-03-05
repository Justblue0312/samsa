"""Context Builder - Gathers and ranks user context signals."""

import logging
from dataclasses import dataclass, field
from typing import Any, Optional

from samsakit.story_recommender.agent.context_signals import (  # type: ignore[import-untyped]
    SignalType,
    ContextPayload,
    create_signal,
    rank_signals,
)

logger = logging.getLogger(__name__)


@dataclass
class ContextBuilder:
    """Builds ranked context from user signals."""

    story_service: Any  # Forward reference
    cache_service: Any  # Forward reference
    max_signals: int = 6

    async def build_context(
        self,
        user_id: str,
        prompt: str,
    ) -> ContextPayload:
        """
        Build context from all available signals.

        Gathers signals in parallel and ranks by effective weight.
        """
        # Collect all signals
        signals = []

        # Signal 1: User prompt (explicit intent)
        signals.append(create_signal(SignalType.USER_PROMPT, {"prompt": prompt}))

        # Signal 2: Bookmarks
        try:
            bookmarks = await self.story_service.get_user_bookmarks(user_id)
            if bookmarks:
                signals.append(
                    create_signal(
                        SignalType.BOOKMARKS,
                        {"bookmarks": bookmarks, "count": len(bookmarks)},
                    )
                )
        except Exception as e:
            logger.warning(f"Failed to fetch bookmarks: {e}")

        # Signal 3: Favorites
        try:
            favorites = await self.story_service.get_user_favorites(user_id)
            if favorites:
                signals.append(
                    create_signal(
                        SignalType.FAVORITES,
                        {"favorites": favorites, "count": len(favorites)},
                    )
                )
        except Exception as e:
            logger.warning(f"Failed to fetch favorites: {e}")

        # Signal 4: Votes (high ratings)
        try:
            votes = await self.story_service.get_user_votes(user_id)
            high_ratings = [v for v in votes if v.get("rating", 0) >= 4]
            if high_ratings:
                signals.append(
                    create_signal(
                        SignalType.VOTES,
                        {"votes": high_ratings, "count": len(high_ratings)},
                    )
                )
        except Exception as e:
            logger.warning(f"Failed to fetch votes: {e}")

        # Signal 5: Recently viewed
        try:
            recently_viewed = await self.story_service.get_recently_viewed(user_id)
            if recently_viewed:
                signals.append(
                    create_signal(
                        SignalType.RECENTLY_VIEWED,
                        {"recently_viewed": recently_viewed, "count": len(recently_viewed)},
                    )
                )
        except Exception as e:
            logger.warning(f"Failed to fetch recently viewed: {e}")

        # Signal 6: Same genre (based on favorites/bookmarks)
        try:
            genres = await self.story_service.get_user_genre_preferences(user_id)
            if genres:
                signals.append(
                    create_signal(
                        SignalType.SAME_GENRE,
                        {"genres": genres},
                    )
                )
        except Exception as e:
            logger.warning(f"Failed to fetch genre preferences: {e}")

        # Rank signals by effective weight
        ranked_signals = rank_signals(signals)

        # Build context payload
        signal_dicts = [
            {
                "type": s.type.value,
                "weight": round(s.effective_weight, 2),
                "summary": self._summarize_signal(s),
            }
            for s in ranked_signals
        ]

        top_signals = signal_dicts[:3]

        return ContextPayload(
            user_id=user_id,
            prompt=prompt,
            signals=signal_dicts,
            top_signals=top_signals,
        )

    def _summarize_signal(self, signal) -> str:
        """Create a human-readable summary of a signal."""
        signal_type = signal.type
        data = signal.data

        if signal_type == SignalType.USER_PROMPT:
            return f"User wants: {data.get('prompt', '')[:50]}"

        if signal_type == SignalType.BOOKMARKS:
            count = data.get("count", 0)
            return f"Bookmarked {count} stories"

        if signal_type == SignalType.FAVORITES:
            count = data.get("count", 0)
            return f"Favorited {count} stories"

        if signal_type == SignalType.VOTES:
            count = data.get("count", 0)
            return f"Highly rated {count} stories"

        if signal_type == SignalType.RECENTLY_VIEWED:
            count = data.get("count", 0)
            return f"Viewed {count} stories recently"

        if signal_type == SignalType.SAME_GENRE:
            genres = data.get("genres", [])
            return f"Interested in: {', '.join(genres[:3])}"

        return str(data)
