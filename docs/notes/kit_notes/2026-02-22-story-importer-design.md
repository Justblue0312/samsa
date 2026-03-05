# Story Importer Agent Design

**Date:** 2026-02-22  
**Status:** Approved  
**Author:** Qwen (with user collaboration)

---

## Overview

This design adds a **Story Importer Agent** that analyzes existing multi-chapter story content, extracts structured world data into Neo4j and Qdrant, and enables interactive clarification when the Writer agent lacks information during generation.

**Problem:** Users want to continue writing stories when they lack ideas, but the AI needs context about characters, plot, and worldbuilding to maintain consistency.

**Solution:** A two-stage workflow:
1. **Import Stage:** Extract full world model from existing chapters
2. **Generation Stage:** Writer agent with interactive clarification for gaps

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    STORY IMPORT WORKFLOW                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  [User Upload Story]                                             │
│         ↓                                                        │
│  ┌─────────────────────┐                                        │
│  │  Importer Agent     │                                        │
│  │  (One-time run)     │                                        │
│  └─────────────────────┘                                        │
│         ↓                                                        │
│    ┌────┴────┐                                                  │
│    ↓         ↓                                                  │
│ ┌──────┐  ┌──────┐                                              │
│ │Neo4j │  │Qdrant│                                              │
│ │Facts │  │Memory│                                              │
│ └──────┘  └──────┘                                              │
│         ↓                                                        │
│  ┌─────────────────────┐                                        │
│  │  Gap Detection      │                                        │
│  │  (Missing info?)    │                                        │
│  └─────────────────────┘                                        │
│         ↓                                                        │
│    [Ask User] ←── Interactive Q&A during write                   │
│         ↓                                                        │
│  ┌─────────────────────┐                                        │
│  │  Writer Agent       │                                        │
│  │  (Task 05)          │                                        │
│  └─────────────────────┘                                        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Location | Purpose |
|-----------|----------|---------|
| **Importer Agent** | `kit/agents/story_importer.py` | Parse story, extract entities, populate databases |
| **Gap Detector** | `kit/graph/clarification.py` | Identify missing world info during generation |
| **Q&A Handler** | `kit/api/clarification_handler.py` | Manage clarification questions to client |
| **Writer Enhancement** | `kit/agents/story_generator.py` | Pause & ask when uncertain |

---

## Neo4j Schema

### Nodes

```cypher
(:Character {
  name: str,
  description: str,
  first_chapter: int,
  traits: list[str],
  goals: list[str]
})

(:Location {
  name: str,
  description: str,
  type: str,  // "city", "building", "region", "fantasy"
  first_mentioned_chapter: int
})

(:Event {
  description: str,
  chapter: int,
  timestamp: str,  // "Day 3", "Year 1024", etc.
  importance: str  // "minor", "major", "climactic"
})

(:Story {
  id: str,
  title: str,
  genre: str,
  tone: str,
  status: str  // "importing", "ready", "completed"
})

(:Timeline {
  chapter: int,
  summary: str,
  word_count: int
})

(:Worldbuilding {
  name: str,
  category: str,  // "magic_system", "technology", "culture", "history"
  description: str,
  rules: list[str]
})

(:Plot {
  arc_name: str,
  type: str,  // "main", "subplot", "character_arc"
  status: str,  // "introduced", "developing", "resolved"
  resolution: str
})

(:PenManner {
  style: str,  // "first_person", "third_limited", "omniscient"
  tense: str,  // "past", "present"
  tone_notes: str,
  prohibited_words: list[str]
})
```

### Relationships

```cypher
// Character relationships
(:Character)-[:FRIEND_OF|ENEMY_OF|LOVER_OF|FAMILY_OF|RIVAL_OF|MENTOR_OF]->(:Character {since_chapter: int})

// Character to location
(:Character)-[:RESIDES_AT|VISITED|ORIGIN_FROM]->(:Location)

// Story containment
(:Story)-[:HAS_CHARACTER]->(:Character)
(:Story)-[:HAS_LOCATION]->(:Location)
(:Story)-[:CONTAINS_EVENT]->(:Event)
(:Story)-[:HAS_TIMELINE]->(:Timeline)
(:Story)-[:HAS_WORLDBUILDING]->(:Worldbuilding)
(:Story)-[:HAS_PLOT]->(:Plot)
(:Story)-[:WRITTEN_IN]->(:PenManner)

// Event connections
(:Event)-[:LEADS_TO]->(:Event)
(:Event)-[:INVOLVES_CHARACTER]->(:Character)
(:Event)-[:OCCURS_AT]->(:Location)

// Plot connections
(:Plot)-[:INVOLVES_CHARACTER]->(:Character)
(:Plot)-[:RESOLVES_IN_EVENT]->(:Event)

// Worldbuilding connections
(:Worldbuilding)-[:AFFECTS_CHARACTER]->(:Character)
(:Worldbuilding)-[:GOVERNS_EVENT]->(:Event)
```

---

## Qdrant Collections

| Collection | Payload Fields | Purpose |
|------------|----------------|---------|
| `story_chapters` | `story_id, chapter_num, section, text, word_count` | Full text chunks for "what happened in chapter X" queries |
| `character_profiles` | `story_id, character_name, traits, goals, relationships_summary` | Quick character lookup for consistency |
| `event_summaries` | `story_id, chapter, event_type, characters_involved, location` | Plot point retrieval |
| `worldbuilding_entries` | `story_id, category, name, rules, description` | Magic systems, tech rules, cultural norms |
| `plot_arcs` | `story_id, arc_name, arc_type, status, involved_characters` | Track subplot progression |
| `clarification_log` | `story_id, question, user_answer, chapter_context, resolved` | History of all Q&A for future reference |

---

## Importer Agent Pipeline

### Step 1: Chapter Parsing
- Split story text by chapter markers
- Generate chapter summaries via LLM
- Create `(:Timeline)` nodes with metadata

### Step 2: Character Extraction
- LLM identifies all named characters
- Extract traits, goals, physical descriptions
- Create `(:Character)` nodes with `first_chapter` tracking

### Step 3: Relationship Extraction
- Analyze character interactions per chapter
- Infer relationship types (friend, enemy, lover, family, rival, mentor)
- Create relationship edges with `since_chapter` metadata

### Step 4: Location Extraction
- Identify all settings, places, geographical elements
- Categorize by type (city, building, region, fantasy)
- Create `(:Location)` nodes with `first_mentioned_chapter`

### Step 5: Event Extraction
- Extract key plot points per chapter
- Rank importance (minor, major, climactic)
- Create `(:Event)` nodes with `LEADS_TO` chains for causality

### Step 6: Worldbuilding Extraction
- Identify magic systems, technology, cultures, history
- Extract explicit rules and constraints
- Create `(:Worldbuilding)` nodes categorized by domain

### Step 7: Plot Arc Extraction
- Identify main plot and all subplots
- Track arc status (introduced, developing, resolved)
- Create `(:Plot)` nodes linked to involved characters

### Step 8: Pen Manner Analysis
- Detect POV style (first person, third limited, omniscient)
- Detect tense (past, present)
- Identify tone patterns and prohibited/repeated words
- Create `(:PenManner)` node as writing constraint reference

### Step 9: Qdrant Population
- Embed and store all 6 collections
- Link payload to Neo4j node IDs for cross-reference queries

### Step 10: Gap Detection
- Check for missing antagonist
- Check for undefined magic/tech rules
- Check for unresolved plot threads
- Generate initial clarification questions

---

## Interactive Clarification Flow

### When Writer Agent Lacks Ideas

```
Writer Node → Detects uncertainty (e.g., "Should John confront Mary now?")
     ↓
Gap Detector → Checks Neo4j/Qdrant for guidance
     ↓
No clear answer → Pauses generation
     ↓
Returns to Client: {
  "status": "clarification_needed",
  "question": "John just discovered Mary's secret. Should he:",
  "options": [
    "A) Confront her immediately (aggressive)",
    "B) Secretly investigate further (cautious)",
    "C) Tell someone else first (seeking advice)"
  ],
  "context": "Chapter 5, Scene 2 - John found the letter"
}
     ↓
User responds → "B"
     ↓
Writer resumes with: "John decided to investigate further before confronting Mary..."
```

### State Management

Add to `StoryState` (Task 04):
```python
class StoryState(TypedDict):
    # Existing fields...
    pending_question: Optional[str]  # Current unanswered question
    clarification_history: list[dict]  # All Q&A pairs
    clarification_options: list[str]  # Multiple choice options
    chapter_context: str  # Where in story the question occurs
```

---

## RPC API Extensions

### Add to `kit/v1/service.proto`

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

// Enhanced GenerateStory for clarification support
message GenerateStoryRequest {
  string story_id = 1;
  string user_id = 2;
  string prompt = 3;
  bool allow_clarification = 4;  // If true, can pause for questions
}
```

---

## Error Handling

| Scenario | Handling |
|----------|----------|
| Import fails mid-process | Rollback Neo4j/Qdrant transactions, retry with checkpoint |
| User doesn't answer clarification | Timeout after configurable duration, use AI default choice with warning |
| Contradictory user answers | Flag for manual review, alert user of inconsistency |
| Very long story (>100 chapters) | Batch import by chapter groups (10 chapters per batch) |
| Multiple genres/tones detected | Ask user to confirm primary genre before import completes |
| Neo4j connection lost | Retry with exponential backoff, queue for later sync |
| Qdrant embedding fails | Fallback to text-only storage, retry embedding later |

---

## File Structure

```
kit/
├── src/
│   └── kit/
│       ├── agents/
│       │   ├── story_importer.py      # NEW: Import pipeline
│       │   ├── story_generator.py     # MODIFIED: Add clarification check
│       │   ├── analyst.py
│       │   ├── content_validator.py
│       │   └── submission_approver.py
│       ├── graph/
│       │   ├── state.py               # MODIFIED: Add clarification fields
│       │   ├── workflow.py
│       │   └── clarification.py       # NEW: Gap detection logic
│       ├── services/
│       │   ├── postgres.py
│       │   ├── neo4j.py               # MODIFIED: Add importer queries
│       │   ├── qdrant.py              # MODIFIED: Add collection methods
│       │   └── telemetry.py
│       ├── api/
│       │   ├── server.py
│       │   ├── handlers.py
│       │   └── clarification_handler.py  # NEW: Q&A endpoint
│       └── gen/
│           └── v1/                    # Generated from proto
└── docs/
    └── plans/kit/
        └── 2026-02-22-story-importer-design.md  # This file
```

---

## Success Criteria

- [ ] Importer successfully extracts all 8 Neo4j node types from sample story
- [ ] All 6 Qdrant collections populated with searchable embeddings
- [ ] Writer agent can pause and request clarification during generation
- [ ] User answers are stored in `clarification_log` and referenced in future generation
- [ ] RPC methods `ImportStory`, `AnswerClarification`, `GetPendingQuestions` functional
- [ ] Gap detection identifies missing antagonist, undefined rules, unresolved plots
- [ ] Import completes within 5 minutes for 50,000-word story

---

## Out of Scope (Future Work)

- **Manual Review Queue:** For contradictory user answers or flagged content
- **Collaborative Import:** Multiple users contributing to same story world
- **Version Control:** Track changes to world model over time
- **Export Function:** Generate story bible document from Neo4j/Qdrant data
- **Visual Graph Editor:** UI for users to manually edit Neo4j relationships

---

## Next Steps

1. **Update Task 01** - Add new RPC methods to proto definition
2. **Create Task 05.5** - Story Importer Agent as new task between Task 05 and Task 06
3. **Update Task 04** - Extend `StoryState` with clarification fields
4. **Update Task 05** - Add `check_uncertainty` node to Writer workflow
5. **Update Task 10** - Add metrics for import duration, clarification frequency
