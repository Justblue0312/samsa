# Story Importer Agent Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Story Importer Agent that extracts structured world data from existing multi-chapter stories into Neo4j and Qdrant, enabling the Writer agent to continue stories with interactive clarification when lacking information.

**Architecture:** Two-stage workflow: (1) Importer Agent parses story content and populates Neo4j/Qdrant with 8 node types and 6 collections, (2) Writer Agent enhanced with gap detection that pauses generation to ask users clarification questions interactively.

**Tech Stack:** Python 3.13, FastAPI, Connect-RPC, LangGraph, Neo4j async driver, Qdrant async client, PydanticAI/OpenRouter, SQLAlchemy+asyncpg, structlog, OpenTelemetry.

---

## Task 0: Update Proto Definitions (Task 01 Extension)

**Files:**
- Modify: `kit/v1/service.proto`
- Modify: `buf.gen.yaml`
- Test: `server/internal/gen/service.pb.go` (verify regeneration)

**Step 1: Add new message types to service.proto**

```protobuf
// Import existing story content for continuation
message ImportStoryRequest {
  string story_id = 1;
  string user_id = 2;
  string title = 3;
  string genre = 4;
  string content = 5;  // Full multi-chapter text
}

message ImportStoryResponse {
  string story_id = 1;
  string status = 2;  // "importing", "ready", "failed"
  repeated string gaps = 3;  // Identified missing information
  repeated string clarification_questions = 4;
}

// Answer a clarification question during generation
message AnswerClarificationRequest {
  string story_id = 1;
  string user_id = 2;
  string question_id = 3;
  string answer = 4;  // User's text response or option selection
}

message AnswerClarificationResponse {
  string story_id = 1;
  bool resumed = 2;  // Whether generation resumed
  string next_context = 3;  // Brief summary of what happens next
}

// Get pending clarification questions
message GetPendingQuestionsRequest {
  string story_id = 1;
  string user_id = 2;
}

message GetPendingQuestionsResponse {
  repeated Question questions = 1;
}

message Question {
  string id = 1;
  string text = 2;
  repeated string options = 3;  // Multiple choice, empty if open-ended
  string chapter_context = 4;
}
```

**Step 2: Update GenerateStoryRequest to support clarification**

```protobuf
message GenerateStoryRequest {
  string story_id = 1;
  string user_id = 2;
  string prompt = 3;
  bool allow_clarification = 4;  // If true, can pause for questions
}
```

**Step 3: Add new RPC services**

```protobuf
service StoryImportService {
  // Import existing story content
  rpc ImportStory(ImportStoryRequest) returns (ImportStoryResponse);
  
  // Answer clarification question
  rpc AnswerClarification(AnswerClarificationRequest) returns (AnswerClarificationResponse);
  
  // Get pending clarification questions
  rpc GetPendingQuestions(GetPendingQuestionsRequest) returns (GetPendingQuestionsResponse);
}
```

**Step 4: Run buf generate**

```bash
buf generate
```

Expected: Generated Python stubs in `kit/src/kit/gen/v1/`, Go stubs in `server/internal/gen/`

**Step 5: Verify Python imports work**

```python
from kit.gen.v1 import service_pb2
req = service_pb2.ImportStoryRequest(story_id="test", content="Chapter 1...")
print(req.story_id)  # Should print "test"
```

**Step 6: Commit**

```bash
git add kit/v1/service.proto kit/src/kit/gen/
git commit -m "proto: add StoryImportService RPC definitions"
```

---

## Task 1: Extend StoryState with Clarification Fields (Task 04 Extension)

**Files:**
- Modify: `kit/src/kit/graph/state.py`
- Test: `kit/tests/graph/test_state.py`

**Step 1: Write test for StoryState with clarification fields**

```python
# kit/tests/graph/test_state.py
from kit.graph.state import StoryState

def test_story_state_includes_clarification_fields():
    state: StoryState = {
        "story_id": "test-123",
        "user_id": "user-456",
        "current_draft": "",
        "feedback": "",
        "safety_flags": [],
        "sentiment": {},
        "iterations": 0,
        # New clarification fields
        "pending_question": None,
        "clarification_history": [],
        "clarification_options": [],
        "chapter_context": "",
    }
    assert state["pending_question"] is None
    assert isinstance(state["clarification_history"], list)
```

**Step 2: Run test to verify it fails**

```bash
uv run pytest kit/tests/graph/test_state.py::test_story_state_includes_clarification_fields -v
```

Expected: FAIL - type error on new fields not in TypedDict

**Step 3: Update StoryState TypedDict**

```python
# kit/src/kit/graph/state.py
from typing import Annotated, List, Optional, TypedDict

class StoryState(TypedDict):
    """The shared memory of the AI story generation workflow."""
    story_id: str
    user_id: str
    current_draft: str
    feedback: str
    safety_flags: List[str]
    sentiment: dict
    iterations: int
    
    # Clarification fields for interactive Q&A
    pending_question: Optional[str]  # Current unanswered question
    clarification_history: List[dict]  # All Q&A pairs
    clarification_options: List[str]  # Multiple choice options
    chapter_context: str  # Where in story the question occurs
```

**Step 4: Run test to verify it passes**

```bash
uv run pytest kit/tests/graph/test_state.py::test_story_state_includes_clarification_fields -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add kit/src/kit/graph/state.py kit/tests/graph/test_state.py
git commit -m "feat: add clarification fields to StoryState"
```

---

## Task 2: Create Neo4j Service Methods for Importer

**Files:**
- Modify: `kit/src/kit/services/neo4j.py`
- Test: `kit/tests/services/test_neo4j.py`

**Step 1: Write test for character creation**

```python
# kit/tests/services/test_neo4j.py
import pytest
from kit.services.neo4j import Neo4jService

@pytest.mark.asyncio
async def test_create_character(neo4j_service):
    result = await neo4j_service.create_character(
        story_id="test-123",
        name="John",
        description="A brave warrior",
        first_chapter=1,
        traits=["brave", "loyal"],
        goals=["defeat the dragon"]
    )
    assert result["name"] == "John"
    assert result["first_chapter"] == 1
```

**Step 2: Run test to verify it fails**

```bash
uv run pytest kit/tests/services/test_neo4j.py::test_create_character -v
```

Expected: FAIL - method not found

**Step 3: Add character creation method**

```python
# kit/src/kit/services/neo4j.py
from typing import List, Optional

class Neo4jService:
    # ... existing init ...
    
    async def create_character(
        self,
        story_id: str,
        name: str,
        description: str,
        first_chapter: int,
        traits: List[str],
        goals: List[str]
    ) -> dict:
        """Create a Character node linked to a Story."""
        query = """
        MATCH (s:Story {id: $story_id})
        CREATE (c:Character {
            name: $name,
            description: $description,
            first_chapter: $first_chapter,
            traits: $traits,
            goals: $goals
        })
        CREATE (s)-[:HAS_CHARACTER]->(c)
        RETURN c {.*, id: elementId(c)} as character
        """
        result = await self.session.run(
            query,
            story_id=story_id,
            name=name,
            description=description,
            first_chapter=first_chapter,
            traits=traits,
            goals=goals
        )
        record = await result.single()
        return record["character"] if record else None
```

**Step 4: Add remaining node creation methods**

Add these methods to `Neo4jService` class:
- `create_relationship()` - Character relationships
- `create_location()` - Location nodes
- `create_event()` - Event nodes
- `create_worldbuilding()` - Worldbuilding elements
- `create_plot_arc()` - Plot arcs
- `create_pen_manner()` - Writing style
- `create_timeline()` - Chapter timeline

**Step 5: Run tests to verify all methods pass**

```bash
uv run pytest kit/tests/services/test_neo4j.py -v
```

Expected: All tests PASS

**Step 6: Commit**

```bash
git add kit/src/kit/services/neo4j.py kit/tests/services/test_neo4j.py
git commit -m "feat: add Neo4j importer methods for all node types"
```

---

## Task 3: Create Qdrant Service Methods for Importer

**Files:**
- Modify: `kit/src/kit/services/qdrant.py`
- Test: `kit/tests/services/test_qdrant.py`

**Step 1: Write test for story chapter storage**

```python
# kit/tests/services/test_qdrant.py
import pytest
from kit.services.qdrant import QdrantService

@pytest.mark.asyncio
async def test_store_chapter(qdrant_service):
    result = await qdrant_service.store_chapter(
        story_id="test-123",
        chapter_num=1,
        section="opening",
        text="Once upon a time...",
        word_count=1500,
        embedding=[0.1] * 1536
    )
    assert result["chapter_num"] == 1
```

**Step 2: Add collection initialization and storage methods**

Add to `QdrantService`:
- `initialize_collections()` - Create 6 collections
- `store_chapter()` - Story chapters
- `store_character_profile()` - Character profiles
- `store_event_summary()` - Event summaries
- `store_worldbuilding_entry()` - Worldbuilding
- `store_plot_arc()` - Plot arcs
- `store_clarification()` - Q&A log

**Step 3: Run tests to verify all methods pass**

```bash
uv run pytest kit/tests/services/test_qdrant.py -v
```

Expected: All tests PASS

**Step 4: Commit**

```bash
git add kit/src/kit/services/qdrant.py kit/tests/services/test_qdrant.py
git commit -m "feat: add Qdrant importer methods for all collections"
```

---

## Task 4: Create Story Importer Agent

**Files:**
- Create: `kit/src/kit/agents/story_importer.py`
- Test: `kit/tests/agents/test_story_importer.py`

**Step 1: Create StoryImporter class with extraction pipeline**

```python
# kit/src/kit/agents/story_importer.py
import re
from typing import List, Dict, Any
from pydantic_ai import Agent

class StoryImporter:
    """Agent that imports existing story content into Neo4j and Qdrant."""
    
    def __init__(
        self,
        neo4j_service: Neo4jService,
        qdrant_service: QdrantService,
        llm_agent: Agent
    ):
        self.neo4j = neo4j_service
        self.qdrant = qdrant_service
        self.llm = llm_agent
    
    async def parse_chapters(self, content: str) -> List[Dict]:
        """Split story content into chapters."""
        # Parse "Chapter N: Title" patterns
        pass
    
    async def extract_characters(self, story_id: str, chapters: List[Dict]) -> List[Dict]:
        """Extract characters using LLM."""
        pass
    
    async def extract_relationships(self, story_id: str, characters: List[Dict], chapters: List[Dict]) -> List[Dict]:
        """Extract character relationships."""
        pass
    
    async def extract_locations(self, story_id: str, chapters: List[Dict]) -> List[Dict]:
        """Extract locations."""
        pass
    
    async def extract_events(self, story_id: str, chapters: List[Dict]) -> List[Dict]:
        """Extract plot events per chapter."""
        pass
    
    async def extract_worldbuilding(self, story_id: str, chapters: List[Dict]) -> List[Dict]:
        """Extract magic, tech, culture, history."""
        pass
    
    async def extract_plot_arcs(self, story_id: str, chapters: List[Dict]) -> List[Dict]:
        """Extract main plot and subplots."""
        pass
    
    async def extract_pen_manner(self, story_id: str, chapters: List[Dict]) -> Dict:
        """Analyze writing style (POV, tense, tone)."""
        pass
    
    async def create_timeline(self, story_id: str, chapters: List[Dict]) -> List[Dict]:
        """Create timeline nodes."""
        pass
    
    async def populate_qdrant(self, story_id: str, chapters: List[Dict], characters: List[Dict], events: List[Dict]):
        """Populate all Qdrant collections."""
        pass
    
    async def detect_gaps(self, story_id: str, characters: List[Dict], events: List[Dict], worldbuilding: List[Dict]) -> List[str]:
        """Detect missing information needing clarification."""
        pass
    
    async def import_story(self, story_id: str, user_id: str, title: str, genre: str, content: str) -> Dict[str, Any]:
        """Main entry point: import entire story."""
        pass
```

**Step 2: Write tests for each extraction method**

**Step 3: Run tests and implement until passing**

```bash
uv run pytest kit/tests/agents/test_story_importer.py -v
```

**Step 4: Commit**

```bash
git add kit/src/kit/agents/story_importer.py kit/tests/agents/test_story_importer.py
git commit -m "feat: create Story Importer agent with full extraction pipeline"
```

---

## Task 5: Create Clarification Handler API

**Files:**
- Create: `kit/src/kit/api/clarification_handler.py`
- Modify: `kit/src/kit/api/handlers.py`
- Create: `kit/migrations/001_create_clarification_questions.sql`
- Test: `kit/tests/api/test_clarification.py`

**Step 1: Create database migration**

```sql
-- kit/migrations/001_create_clarification_questions.sql

CREATE TABLE IF NOT EXISTS clarification_questions (
    id SERIAL PRIMARY KEY,
    story_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    question_id VARCHAR(255) NOT NULL,
    text TEXT NOT NULL,
    options TEXT[] DEFAULT '{}',
    chapter_context TEXT,
    answer TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    answered_at TIMESTAMP,
    UNIQUE(story_id, question_id)
);

CREATE INDEX idx_clarification_story ON clarification_questions(story_id);
CREATE INDEX idx_clarification_status ON clarification_questions(status);
```

**Step 2: Create ClarificationHandler class**

```python
# kit/src/kit/api/clarification_handler.py
from typing import List, Dict, Optional

class ClarificationHandler:
    """Handles clarification questions between Writer agent and user."""
    
    def __init__(self, postgres: PostgresService):
        self.db = postgres
    
    async def add_question(self, story_id: str, question_id: str, text: str, options: List[str], chapter_context: str) -> Dict:
        """Store a pending clarification question."""
        pass
    
    async def get_pending_questions(self, story_id: str, user_id: str) -> List[Dict]:
        """Get all pending questions for a story."""
        pass
    
    async def answer_question(self, story_id: str, question_id: str, answer: str) -> Dict:
        """Record user's answer."""
        pass
    
    async def get_answer(self, story_id: str, question_id: str) -> Optional[str]:
        """Get user's answer."""
        pass
```

**Step 3: Add RPC handlers to handlers.py**

- `ImportStory()` - Call StoryImporter
- `AnswerClarification()` - Record user answer
- `GetPendingQuestions()` - Return pending questions

**Step 4: Write and run tests**

```bash
uv run pytest kit/tests/api/test_clarification.py -v
```

**Step 5: Commit**

```bash
git add kit/src/kit/api/clarification_handler.py kit/src/kit/api/handlers.py kit/migrations/001_create_clarification_questions.sql kit/tests/api/test_clarification.py
git commit -m "feat: add ClarificationHandler API for interactive Q&A"
```

---

## Task 6: Enhance Writer Agent with Clarification Check

**Files:**
- Modify: `kit/src/kit/agents/story_generator.py`
- Modify: `kit/src/kit/graph/workflow.py`
- Test: `kit/tests/agents/test_story_generator.py`

**Step 1: Add check_uncertainty node**

```python
# kit/src/kit/agents/story_generator.py

async def check_uncertainty(state: Dict[str, Any]) -> Dict[str, Any]:
    """Detect if generation needs clarification from user."""
    # Use LLM to analyze draft for uncertainty
    # Return needs_clarification, question, options
    pass

async def pause_for_clarification(state: Dict[str, Any]) -> Dict[str, Any]:
    """Pause generation and store question in database."""
    pass

async def resume_generation(state: Dict[str, Any]) -> Dict[str, Any]:
    """Resume generation after user answers."""
    pass
```

**Step 2: Update workflow with clarification edges**

```python
# kit/src/kit/graph/workflow.py

workflow.add_node("uncertainty_check", check_uncertainty)
workflow.add_node("pause_for_clarification", pause_for_clarification)

workflow.add_conditional_edges(
    "uncertainty_check",
    should_continue,
    {
        "ask_user": "pause_for_clarification",
        "continue": "validator",
        "finish": END
    }
)
```

**Step 3: Write and run tests**

```bash
uv run pytest kit/tests/agents/test_story_generator.py -v
```

**Step 4: Commit**

```bash
git add kit/src/kit/agents/story_generator.py kit/src/kit/graph/workflow.py kit/tests/agents/test_story_generator.py
git commit -m "feat: add clarification check to Writer agent workflow"
```

---

## Task 7: Add LLM Service Configuration

**Files:**
- Create: `kit/src/kit/services/llm.py`
- Modify: `kit/src/kit/settings.py`
- Test: `kit/tests/services/test_llm.py`

**Step 1: Create LLM service**

```python
# kit/src/kit/services/llm.py
from pydantic_ai import Agent

def get_llm(model: str = None) -> Agent:
    """Get configured LLM agent for story tasks."""
    pass

def get_embedding_model():
    """Get embedding model for Qdrant."""
    pass
```

**Step 2: Add settings**

```python
# kit/src/kit/settings.py
LLM_MODEL: str = "openrouter:meta-llama/llama-3.1-70b-instruct"
OPENROUTER_API_KEY: str = ""
EMBEDDING_MODEL: str = "openrouter:text-embedding-3-large"
```

**Step 3: Write and run tests**

```bash
uv run pytest kit/tests/services/test_llm.py -v
```

**Step 4: Commit**

```bash
git add kit/src/kit/services/llm.py kit/src/kit/settings.py kit/tests/services/test_llm.py
git commit -m "feat: add centralized LLM configuration service"
```

---

## Task 8: Integration Testing

**Files:**
- Create: `kit/tests/integration/test_importer_e2e.py`

**Step 1: Write end-to-end import test**

Test full import flow with sample story, verify Neo4j nodes and Qdrant collections populated.

**Step 2: Run integration test**

```bash
uv run pytest kit/tests/integration/test_importer_e2e.py -v --integration
```

**Step 3: Commit**

```bash
git add kit/tests/integration/test_importer_e2e.py
git commit -m "test: add end-to-end integration test for Story Importer"
```

---

## Task 9: Documentation Update

**Files:**
- Modify: `docs/plans/kit/IMPLEMENTATION-CHECKLISTS.md`
- Create: `docs/plans/kit/STORY-IMPORTER-GUIDE.md`

**Step 1: Add Task 05.5 checklist**

Add Story Importer section with extraction pipeline checklist.

**Step 2: Create user guide**

Include usage examples, API endpoints, best practices.

**Step 3: Commit**

```bash
git add docs/plans/kit/IMPLEMENTATION-CHECKLISTS.md docs/plans/kit/STORY-IMPORTER-GUIDE.md
git commit -m "docs: add Story Importer documentation"
```

---

## Task 10: Final Verification & Cleanup

**Step 1: Run full test suite**

```bash
uv run pytest kit/tests/ -v --tb=short
```

**Step 2: Run linting**

```bash
uv run ruff check kit/src/
```

**Step 3: Run type checking**

```bash
uv run mypy kit/src/
```

**Step 4: Verify Docker build**

```bash
docker build -t fiente-kit:latest -f kit/Dockerfile kit/
```

**Step 5: Commit final changes**

```bash
git add .
git commit -m "chore: final verification and cleanup for Story Importer"
```

---

## Summary

**Total Tasks:** 10

**Files Created:**
- `kit/src/kit/agents/story_importer.py`
- `kit/src/kit/api/clarification_handler.py`
- `kit/src/kit/services/llm.py`
- `kit/migrations/001_create_clarification_questions.sql`
- `docs/plans/kit/STORY-IMPORTER-GUIDE.md`

**Files Modified:**
- `kit/v1/service.proto`
- `kit/src/kit/graph/state.py`
- `kit/src/kit/services/neo4j.py`
- `kit/src/kit/services/qdrant.py`
- `kit/src/kit/api/handlers.py`
- `kit/src/kit/agents/story_generator.py`
- `kit/src/kit/graph/workflow.py`
- `kit/src/kit/settings.py`
- `docs/plans/kit/IMPLEMENTATION-CHECKLISTS.md`

---

Plan complete and saved to `docs/plans/kit\story-importer-implementation.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
