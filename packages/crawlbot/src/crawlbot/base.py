from abc import ABC, abstractmethod
from dataclasses import dataclass


class BaseStory(ABC):
    def __init__(self): ...

    @abstractmethod
    def get_site_name(self) -> str: ...


@dataclass
class Story:
    title: str
    description: str | None = None
    max_chapters: int | None = None


@dataclass
class Chapter:
    title: str
    content: str
    chapter_number: int | None = None
