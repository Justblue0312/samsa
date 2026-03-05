# Task 08: Agent - Submission Approver (The "Gatekeeper")

## Overview
This task involves building the final decision agent that combines inputs from the Story Analyst and Content Validator to automate (or escalate) submission approvals.

## Objectives
*   Implement an automated decision-maker based on user reputation and content scores.
*   Route submissions to "Auto-Approve," "Auto-Reject," or "Manual Review" categories.
*   Integrate with the `SubmissionService` (Go) via the `AgentService.ValidateContent` RPC.

## Components
*   **Approver Node (`kit/agents/submission_approver.py`):**
    *   `reputation_check(user_id)` node: Fetches the user's historical approval rate from Postgres.
    *   `content_score_threshold(sentiment, consistency, safety_flags)` node: Combines the outputs of previous nodes.
    *   `final_decision(approval_score)` node: Categorizes the submission.
*   **Decision Logic:** 
    *   If user reputation is high AND Content Validator finds no flags -> Auto-Approve.
    *   If Content Validator finds critical flags -> Auto-Reject.
    *   If scores are borderline -> Manual Review.

## Success Criteria
*   High-quality submissions from trusted users are automatically approved without human intervention.
*   Low-quality or prohibited submissions are automatically rejected with clear reasoning.
*   The Go backend receives the `Approved`/`Rejected` status along with all metadata correctly.

## Next Steps
*   Perform full integration and end-to-end testing in Task 09.
