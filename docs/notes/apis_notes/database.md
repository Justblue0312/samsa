# Fiente: Unified SaaS Database Design (v2)

This document defines the complete database schema for Fiente, integrating existing core features with new SaaS capabilities centered around **Workspaces (Organizations)** and **Scope-based RBAC**.

## 1. Core Identity & Global Access

We use a **Scope-based system** instead of complex Role/Permission tables. Scopes are simple strings (e.g., `story:create`, `sys:admin`).

### Existing Tables:

- **`users`**: Primary identity.
    - `id`, `email`, `password_hash`, `is_active`, `global_scopes` (JSONB/String Array).
- **`sessions`**: Active login tracking.
- **`oauth_accounts`**: Third-party provider links (Google, GitHub).

### New SaaS Layer:

- **`user_settings`**: UI preferences, editor defaults.

---

## 2. Organizations (Workspaces)

The **Workspace** is treated as a primary actor (The Organization). All professional activity happens within a Workspace.

### New Tables:

- **`workspaces`**: The "Organization" entity.
    - `id`, `name`, `slug` (unique), `owner_id`, `plan_id`, `created_at`.
- **`workspace_members`**: Links users to organizations.
    - `id`, `workspace_id`, `user_id`, `scopes` (JSONB/Array of local scopes).
    - _Example Scopes_: `member:owner`, `content:editor`, `billing:manager`.

---

## 3. Content & Production (Core Restored)

Content belongs to a **Workspace** to support multi-tenancy and team collaboration.

### Existing Tables (Updated for Workspaces):

- **`stories`**: The top-level creative work.
    - `id`, `workspace_id` (added), `author_id`, `title`, `description`, `status`.
- **`chapters`**: Structural divisions of a story.
- **`documents`**: The actual text content/drafts.
- **`document_folders`**: Logical organization for documents.
- **`authors`**: Professional profiles (can be linked to a user/workspace).

### New Features:

- **`story_versions`**: Snapshots for "time travel" and data safety.
- **`files`**: Assets (covers, character art) now linked to `workspace_id`.

---

## 4. Taxonomy & Metadata (Core Restored)

Shared across the platform for discovery.

### Existing Tables:

- **`genres`**, **`story_genres`**: High-level classification.
- **`tags`**, **`story_tags`**: Granular discovery labels.

---

## 5. Social & Engagement (Core Restored)

User-to-User and User-to-Content interactions.

### Existing Tables:

- **`comments`**, **`comment_reactions`**, **`comment_votes`**.
- **`story_votes`**, **`story_reports`**, **`flags`**.
- **`user_bookmarks`**, **`user_favorites`**, **`user_follows`**.
- **`notifications`**, **`user_notification_preferences`**.

---

## 6. Monetization & SaaS Operations

### New Tables:

- **`plans`**: Definition of tiers (Free, Pro, Team).
    - `id`, `name`, `price_monthly`, `features` (JSONB).
- **`subscriptions`**: Active status of a **Workspace**.
    - `id`, `workspace_id`, `plan_id`, `status`, `current_period_end`.
- **`billing_history`**: Invoices and payments for a Workspace.

---

## 7. Staff & Governance

### New Tables:

- **`audit_logs`**: Forensic trail of actions.
    - `id`, `actor_id` (User), `workspace_id` (Optional), `action`, `metadata`.
- **`moderation_queues`**: Unified workflow for staff to resolve reports/flags.

---

## Summary of Actor Scopes

| Actor                | Scope Level          | Examples                                           |
| :------------------- | :------------------- | :------------------------------------------------- |
| **Individual User**  | Global               | `sys:mod`, `sys:admin`, `profile:edit`             |
| **Workspace Member** | Local (to Workspace) | `content:write`, `content:publish`, `billing:view` |
| **Staff/Admin**      | Global               | `user:ban`, `plan:manage`, `audit:read`            |

## Implementation Strategy

1. **Migrations**: Add `workspace_id` to `stories`, `files`, and `authors`.
2. **Context**: Pass `workspace_id` in the Go backend `context.Context` to ensure tenant isolation.
3. **Logic**: Use a `HasScope(ctx, "content:write")` helper for all actions.
