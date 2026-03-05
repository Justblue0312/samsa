# Task 07: Agent - Content Validator (The "Moderator")

## Overview
This task involves building the safety and policy enforcement agent that ensures content complies with the project's guidelines.

## Objectives
*   Implement a rule-based and LLM-powered safety checker.
*   Check for prohibited themes, hate speech, and platform-specific policy violations.
*   Integrate with the `ValidateContent` RPC response.

## Components
*   **Validator Node (`kit/agents/content_validator.py`):**
    *   `rule_check(content)` node: Uses regex and keyword matching for fast, deterministic checks.
    *   `llm_safety_check(content)` node: Uses an LLM with a specialized "Safety System Prompt" to evaluate nuanced policy issues.
    *   `flag_violation(reason, category)` node: Adds flags to the `StoryState`.
*   **Safety Thresholds:** Configure sensitivity levels for different story genres.

## Success Criteria
*   The validator correctly flags prohibited content (e.g., hate speech, graphic violence not allowed in the genre).
*   The reasoning for rejection is clear and helpful to the user.
*   The validator is robust enough to handle adversarial prompts.

## Next Steps
*   Implement the `Submission Approver Agent` in Task 08.
