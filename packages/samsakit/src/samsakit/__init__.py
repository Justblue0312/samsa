"""Samsa Python Toolkit."""

from .story_recommender import main as story_recommender_main

__version__ = "0.1.0"


def run_cli() -> None:
    """Entry point for the samsakit CLI."""
    print("Samsa Python Toolkit")
    print("Run: python -m samsakit.story_recommender.main")
