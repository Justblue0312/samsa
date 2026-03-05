from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RecommendationRequest(_message.Message):
    __slots__ = ("user_id", "prompt", "limit")
    USER_ID_FIELD_NUMBER: _ClassVar[int]
    PROMPT_FIELD_NUMBER: _ClassVar[int]
    LIMIT_FIELD_NUMBER: _ClassVar[int]
    user_id: str
    prompt: str
    limit: int
    def __init__(self, user_id: _Optional[str] = ..., prompt: _Optional[str] = ..., limit: _Optional[int] = ...) -> None: ...

class StoryRecommendation(_message.Message):
    __slots__ = ("story_id", "title", "synopsis", "author", "genres", "reason", "signal_source", "confidence_note")
    STORY_ID_FIELD_NUMBER: _ClassVar[int]
    TITLE_FIELD_NUMBER: _ClassVar[int]
    SYNOPSIS_FIELD_NUMBER: _ClassVar[int]
    AUTHOR_FIELD_NUMBER: _ClassVar[int]
    GENRES_FIELD_NUMBER: _ClassVar[int]
    REASON_FIELD_NUMBER: _ClassVar[int]
    SIGNAL_SOURCE_FIELD_NUMBER: _ClassVar[int]
    CONFIDENCE_NOTE_FIELD_NUMBER: _ClassVar[int]
    story_id: str
    title: str
    synopsis: str
    author: str
    genres: _containers.RepeatedScalarFieldContainer[str]
    reason: str
    signal_source: str
    confidence_note: str
    def __init__(self, story_id: _Optional[str] = ..., title: _Optional[str] = ..., synopsis: _Optional[str] = ..., author: _Optional[str] = ..., genres: _Optional[_Iterable[str]] = ..., reason: _Optional[str] = ..., signal_source: _Optional[str] = ..., confidence_note: _Optional[str] = ...) -> None: ...

class RecommendationResponse(_message.Message):
    __slots__ = ("event", "recommendation", "cached_results", "token_usage", "signals", "error", "total", "elapsed_ms")
    EVENT_FIELD_NUMBER: _ClassVar[int]
    RECOMMENDATION_FIELD_NUMBER: _ClassVar[int]
    CACHED_RESULTS_FIELD_NUMBER: _ClassVar[int]
    TOKEN_USAGE_FIELD_NUMBER: _ClassVar[int]
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    TOTAL_FIELD_NUMBER: _ClassVar[int]
    ELAPSED_MS_FIELD_NUMBER: _ClassVar[int]
    event: str
    recommendation: StoryRecommendation
    cached_results: CachedResults
    token_usage: TokenUsage
    signals: ContextSignals
    error: ErrorMessage
    total: int
    elapsed_ms: int
    def __init__(self, event: _Optional[str] = ..., recommendation: _Optional[_Union[StoryRecommendation, _Mapping]] = ..., cached_results: _Optional[_Union[CachedResults, _Mapping]] = ..., token_usage: _Optional[_Union[TokenUsage, _Mapping]] = ..., signals: _Optional[_Union[ContextSignals, _Mapping]] = ..., error: _Optional[_Union[ErrorMessage, _Mapping]] = ..., total: _Optional[int] = ..., elapsed_ms: _Optional[int] = ...) -> None: ...

class CachedResults(_message.Message):
    __slots__ = ("stories", "note")
    STORIES_FIELD_NUMBER: _ClassVar[int]
    NOTE_FIELD_NUMBER: _ClassVar[int]
    stories: _containers.RepeatedCompositeFieldContainer[StoryRecommendation]
    note: str
    def __init__(self, stories: _Optional[_Iterable[_Union[StoryRecommendation, _Mapping]]] = ..., note: _Optional[str] = ...) -> None: ...

class TokenUsage(_message.Message):
    __slots__ = ("agents", "total_tokens", "total_cost_usd", "total_elapsed_ms")
    class AgentUsage(_message.Message):
        __slots__ = ("name", "total_tokens", "elapsed_ms", "cost_usd")
        NAME_FIELD_NUMBER: _ClassVar[int]
        TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
        ELAPSED_MS_FIELD_NUMBER: _ClassVar[int]
        COST_USD_FIELD_NUMBER: _ClassVar[int]
        name: str
        total_tokens: int
        elapsed_ms: int
        cost_usd: float
        def __init__(self, name: _Optional[str] = ..., total_tokens: _Optional[int] = ..., elapsed_ms: _Optional[int] = ..., cost_usd: _Optional[float] = ...) -> None: ...
    AGENTS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_TOKENS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_COST_USD_FIELD_NUMBER: _ClassVar[int]
    TOTAL_ELAPSED_MS_FIELD_NUMBER: _ClassVar[int]
    agents: _containers.RepeatedCompositeFieldContainer[TokenUsage.AgentUsage]
    total_tokens: int
    total_cost_usd: float
    total_elapsed_ms: int
    def __init__(self, agents: _Optional[_Iterable[_Union[TokenUsage.AgentUsage, _Mapping]]] = ..., total_tokens: _Optional[int] = ..., total_cost_usd: _Optional[float] = ..., total_elapsed_ms: _Optional[int] = ...) -> None: ...

class ContextSignals(_message.Message):
    __slots__ = ("signals",)
    class Signal(_message.Message):
        __slots__ = ("type", "weight", "summary")
        TYPE_FIELD_NUMBER: _ClassVar[int]
        WEIGHT_FIELD_NUMBER: _ClassVar[int]
        SUMMARY_FIELD_NUMBER: _ClassVar[int]
        type: str
        weight: float
        summary: str
        def __init__(self, type: _Optional[str] = ..., weight: _Optional[float] = ..., summary: _Optional[str] = ...) -> None: ...
    SIGNALS_FIELD_NUMBER: _ClassVar[int]
    signals: _containers.RepeatedCompositeFieldContainer[ContextSignals.Signal]
    def __init__(self, signals: _Optional[_Iterable[_Union[ContextSignals.Signal, _Mapping]]] = ...) -> None: ...

class ErrorMessage(_message.Message):
    __slots__ = ("message",)
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    message: str
    def __init__(self, message: _Optional[str] = ...) -> None: ...

class HealthCheckRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class HealthCheckResponse(_message.Message):
    __slots__ = ("status",)
    class ServingStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        SERVING_STATUS_UNSPECIFIED: _ClassVar[HealthCheckResponse.ServingStatus]
        SERVING: _ClassVar[HealthCheckResponse.ServingStatus]
        NOT_SERVING: _ClassVar[HealthCheckResponse.ServingStatus]
    SERVING_STATUS_UNSPECIFIED: HealthCheckResponse.ServingStatus
    SERVING: HealthCheckResponse.ServingStatus
    NOT_SERVING: HealthCheckResponse.ServingStatus
    STATUS_FIELD_NUMBER: _ClassVar[int]
    status: HealthCheckResponse.ServingStatus
    def __init__(self, status: _Optional[_Union[HealthCheckResponse.ServingStatus, str]] = ...) -> None: ...
