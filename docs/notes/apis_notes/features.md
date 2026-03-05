# Samsa Writing Platform - API Feature Improvements

## Overview

This document outlines comprehensive feature improvements to make the Samsa writing platform competitive with modern writing platforms like Wattpad, Medium, and Substack. Each feature area includes complete database schema specifications, API endpoints, and integration mappings to existing tables.

## Database Schema Overview

### Existing Schema (22 Tables)

1. **User Management**: `user`, `session`, `oauth_account`
2. **Author Management**: `author`, `submission`
3. **Story Management**: `story`, `chapter`, `story_status_history`, `story_stats_mv`, `story_vote`, `user_bookmark`, `user_favorite`, `flag`, `story_report`
4. **Content Management**: `file`, `tag`, `document`, `document_version`, `template`, `genre`
5. **Social Features**: `comment`, `comment_reaction`, `notification`, `user_follow`
6. **System**: `activity`, `configuration`
7. **Association Tables**: `story_tag`, `chapter_tag`, `story_genre`, `chapter_document`, `story_flag`, `chapter_comment`, `story_comment`, `submission_comment`

### New Feature Areas (5)

1. **Reading Experience**: 3 new tables
2. **Author Analytics**: 3 new tables
3. **Monetization**: 4 new tables
4. **Collaboration**: 4 new tables
5. **Discovery**: 3 new tables

**Total New Tables**: 17
**Total Tables After Implementation**: 39

---

## 1. Reading Experience Features

### 1.1 Database Schema

#### Table: `reading_progress`

Tracks user's reading progress for each chapter.

```sql
CREATE TABLE reading_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- User and Content References
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID NOT NULL REFERENCES chapter(id) ON DELETE CASCADE,

    -- Reading Position
    scroll_position DECIMAL(10, 2) DEFAULT 0, -- Percentage (0-100)
    last_paragraph_index INTEGER DEFAULT 0,
    total_words_read INTEGER DEFAULT 0,

    -- Reading Status
    is_completed BOOLEAN DEFAULT FALSE,
    completion_percentage DECIMAL(5, 2) DEFAULT 0,

    -- Reading Session Data
    time_spent_seconds INTEGER DEFAULT 0, -- Total time spent reading
    reading_speed_wpm DECIMAL(5, 2), -- Calculated reading speed

    -- Session Tracking
    last_read_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Device/Session Info
    device_type CHAR(50), -- mobile, tablet, desktop
    session_id UUID REFERENCES session(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT reading_progress_user_chapter_unique UNIQUE (user_id, chapter_id),
    CONSTRAINT reading_progress_scroll_range CHECK (scroll_position >= 0 AND scroll_position <= 100),
    CONSTRAINT reading_progress_completion_range CHECK (completion_percentage >= 0 AND completion_percentage <= 100)
);

-- Indexes
CREATE INDEX idx_reading_progress_user_id ON reading_progress(user_id);
CREATE INDEX idx_reading_progress_story_id ON reading_progress(story_id);
CREATE INDEX idx_reading_progress_chapter_id ON reading_progress(chapter_id);
CREATE INDEX idx_reading_progress_last_read ON reading_progress(user_id, last_read_at DESC);
CREATE INDEX idx_reading_progress_in_progress ON reading_progress(user_id, is_completed) WHERE is_completed = FALSE;
CREATE INDEX idx_reading_progress_completed ON reading_progress(user_id, completed_at) WHERE is_completed = TRUE;
```

#### Table: `reading_list`

User-created reading lists/collections.

```sql
CREATE TYPE reading_list_privacy AS ENUM ('private', 'public', 'shared');

CREATE TABLE reading_list (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Owner
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- List Information
    name CHAR(255) NOT NULL,
    description TEXT,
    slug CHAR(255) NOT NULL,

    -- Privacy & Visibility
    privacy reading_list_privacy DEFAULT 'private',

    -- Display Settings
    cover_image_id UUID REFERENCES file(id) ON DELETE SET NULL,
    sort_order INTEGER DEFAULT 0,
    is_default BOOLEAN DEFAULT FALSE, -- System default lists (Read Later, Currently Reading, etc.)

    -- Statistics
    total_stories INTEGER DEFAULT 0,
    total_stories_completed INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT reading_lists_user_slug_unique UNIQUE (user_id, slug)
);

-- Indexes
CREATE INDEX idx_reading_lists_user_id ON reading_list(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reading_lists_privacy ON reading_list(privacy) WHERE deleted_at IS NULL;
CREATE INDEX idx_reading_lists_default ON reading_list(user_id, is_default) WHERE is_default = TRUE AND deleted_at IS NULL;
CREATE INDEX idx_reading_lists_slug ON reading_list(slug) WHERE deleted_at IS NULL;
```

#### Table: `reading_list_item`

Stories within a reading list.

```sql
CREATE TABLE reading_list_item (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    list_id UUID NOT NULL REFERENCES reading_list(id) ON DELETE CASCADE,
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,

    -- Position & Order
    sort_order INTEGER DEFAULT 0,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Reading Status for this List
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP WITH TIME ZONE,
    last_read_chapter_id UUID REFERENCES chapter(id) ON DELETE SET NULL,

    -- User Notes
    notes TEXT,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT reading_list_items_list_story_unique UNIQUE (list_id, story_id)
);

-- Indexes
CREATE INDEX idx_reading_list_items_list_id ON reading_list_item(list_id);
CREATE INDEX idx_reading_list_items_story_id ON reading_list_item(story_id);
CREATE INDEX idx_reading_list_items_completed ON reading_list_item(list_id, is_completed);
```

#### Table: `offline_content`

Tracks content available for offline reading.

```sql
CREATE TABLE offline_content (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE, -- NULL = entire story

    -- Download Info
    file_id UUID REFERENCES file(id) ON DELETE SET NULL, -- Cached file reference
    download_size_bytes BIGINT DEFAULT 0,

    -- Content Version
    content_hash CHAR(64), -- SHA-256 of content
    downloaded_version INTEGER DEFAULT 1,

    -- Sync Status
    is_synced BOOLEAN DEFAULT FALSE,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    sync_error TEXT,

    -- Availability
    is_available BOOLEAN DEFAULT TRUE,
    expires_at TIMESTAMP WITH TIME ZONE, -- Optional expiration

    -- Device Info
    device_id CHAR(255), -- Client device identifier

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT offline_content_user_story_chapter_unique UNIQUE (user_id, story_id, chapter_id)
);

-- Indexes
CREATE INDEX idx_offline_content_user_id ON offline_content(user_id);
CREATE INDEX idx_offline_content_story_id ON offline_content(story_id);
CREATE INDEX idx_offline_content_synced ON offline_content(user_id, is_synced) WHERE is_synced = FALSE;
CREATE INDEX idx_offline_content_available ON offline_content(user_id, is_available) WHERE is_available = TRUE;
```

### 1.2 API Endpoints

#### Reading Progress Endpoints

| Method | Path                                       | Description                            | Auth Required |
| ------ | ------------------------------------------ | -------------------------------------- | ------------- |
| POST   | `/api/v1/reading/progress`                 | Update reading progress                | User          |
| GET    | `/api/v1/reading/progress/{chapterId}`     | Get progress for chapter               | User          |
| GET    | `/api/v1/reading/progress`                 | List all reading progress              | User          |
| GET    | `/api/v1/reading/progress/story/{storyId}` | Get progress for entire story          | User          |
| DELETE | `/api/v1/reading/progress/{chapterId}`     | Reset progress for chapter             | User          |
| GET    | `/api/v1/reading/continue`                 | Get "Continue Reading" recommendations | User          |

#### Reading Lists Endpoints

| Method | Path                                                     | Description              | Auth Required |
| ------ | -------------------------------------------------------- | ------------------------ | ------------- |
| GET    | `/api/v1/reading/lists`                                  | Get user's reading lists | User          |
| POST   | `/api/v1/reading/lists`                                  | Create new reading list  | User          |
| GET    | `/api/v1/reading/lists/{listId}`                         | Get reading list details | User          |
| PUT    | `/api/v1/reading/lists/{listId}`                         | Update reading list      | User          |
| DELETE | `/api/v1/reading/lists/{listId}`                         | Delete reading list      | User          |
| POST   | `/api/v1/reading/lists/{listId}/stories`                 | Add story to list        | User          |
| DELETE | `/api/v1/reading/lists/{listId}/stories/{storyId}`       | Remove story from list   | User          |
| PUT    | `/api/v1/reading/lists/{listId}/stories/{storyId}/order` | Reorder story in list    | User          |
| GET    | `/api/v1/reading/lists/{listId}/stories`                 | Get stories in list      | User          |
| GET    | `/api/v1/reading/lists/public/{slug}`                    | Get public reading list  | No            |

#### Offline Content Endpoints

| Method | Path                                           | Description              | Auth Required |
| ------ | ---------------------------------------------- | ------------------------ | ------------- |
| POST   | `/api/v1/reading/offline`                      | Mark content for offline | User          |
| DELETE | `/api/v1/reading/offline/{contentId}`          | Remove from offline      | User          |
| GET    | `/api/v1/reading/offline`                      | List offline content     | User          |
| POST   | `/api/v1/reading/offline/sync`                 | Sync offline content     | User          |
| GET    | `/api/v1/reading/offline/{contentId}/download` | Download offline content | User          |
| GET    | `/api/v1/reading/offline/status`               | Get sync status          | User          |

### 1.3 Request/Response Examples

#### Update Reading Progress

**Request:**

```json
POST /api/v1/reading/progress
{
  "chapterId": "uuid",
  "scrollPosition": 45.5,
  "lastParagraphIndex": 23,
  "totalWordsRead": 1250,
  "timeSpentSeconds": 300,
  "isCompleted": false,
  "completionPercentage": 45.5,
  "deviceType": "mobile"
}
```

**Response:**

```json
{
    "data": {
        "id": "uuid",
        "chapterId": "uuid",
        "storyId": "uuid",
        "scrollPosition": 45.5,
        "completionPercentage": 45.5,
        "isCompleted": false,
        "timeSpentSeconds": 300,
        "readingSpeedWpm": 250.0,
        "lastReadAt": "2026-01-14T10:30:00Z",
        "estimatedTimeRemaining": 360
    },
    "message": "Reading progress updated"
}
```

#### Create Reading List

**Request:**

```json
POST /api/v1/reading/lists
{
  "name": "Summer Reading 2026",
  "description": "Best stories to read this summer",
  "privacy": "public",
  "coverImageId": "uuid"
}
```

**Response:**

```json
{
    "data": {
        "id": "uuid",
        "name": "Summer Reading 2026",
        "slug": "summer-reading-2026",
        "description": "Best stories to read this summer",
        "privacy": "public",
        "totalStories": 0,
        "totalStoriesCompleted": 0,
        "createdAt": "2026-01-14T10:30:00Z"
    },
    "message": "Reading list created"
}
```

### 1.4 Integration Points

- **Progress Tracking**: Updates trigger recalculation of `reading_speed_wpm` and story completion stats
- **Reading Lists**: Auto-populate with existing bookmarks/favorites during migration
- **Offline Sync**: Integrates with existing `file` table for content caching
- **Analytics**: Reading progress feeds into author analytics tables

---

## 2. Author Analytics Features

### 2.1 Database Schema

#### Table: `analytics_view`

Detailed view analytics for stories and chapters.

```sql
CREATE TYPE view_source AS ENUM ('direct', 'search', 'social', 'recommendation', 'email', 'rss', 'external');
CREATE TYPE view_device AS ENUM ('desktop', 'mobile', 'tablet', 'unknown');

CREATE TABLE analytics_view (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Content References
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,

    -- Viewer Information
    viewer_id UUID REFERENCES "user"(id) ON DELETE SET NULL, -- NULL = anonymous
    is_unique_view BOOLEAN DEFAULT TRUE,
    is_returning_viewer BOOLEAN DEFAULT FALSE,

    -- View Context
    view_source view_source DEFAULT 'direct',
    referrer_url TEXT,
    landing_page CHAR(255),

    -- Device & Location
    device_type view_device DEFAULT 'unknown',
    device_browser CHAR(100),
    device_os CHAR(100),
    country_code CHAR(2),
    region CHAR(100),
    city CHAR(100),

    -- Engagement Metrics
    time_on_page_seconds INTEGER DEFAULT 0,
    scroll_depth_percentage INTEGER DEFAULT 0,
    did_bounce BOOLEAN DEFAULT TRUE, -- Left without interaction

    -- Session Tracking
    session_id UUID,
    view_sequence_in_session INTEGER DEFAULT 1,

    -- Timestamps
    viewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT analytics_view_scroll_range CHECK (scroll_depth_percentage >= 0 AND scroll_depth_percentage <= 100)
);

-- Indexes
CREATE INDEX idx_analytics_view_story_id ON analytics_view(story_id);
CREATE INDEX idx_analytics_view_chapter_id ON analytics_view(chapter_id) WHERE chapter_id IS NOT NULL;
CREATE INDEX idx_analytics_view_author_id ON analytics_view(author_id);
CREATE INDEX idx_analytics_view_viewed_at ON analytics_view(viewed_at DESC);
CREATE INDEX idx_analytics_view_author_date ON analytics_view(author_id, viewed_at DESC);
CREATE INDEX idx_analytics_view_unique ON analytics_view(story_id, viewer_id, DATE(viewed_at)) WHERE is_unique_view = TRUE;
CREATE INDEX idx_analytics_view_source ON analytics_view(view_source);
CREATE INDEX idx_analytics_view_device ON analytics_view(device_type);
CREATE INDEX idx_analytics_view_country ON analytics_view(country_code);
```

#### Table: `analytics_engagement`

User engagement events (clicks, shares, comments, votes).

```sql
CREATE TYPE engagement_type AS ENUM (
    'vote', 'favorite', 'bookmark', 'comment', 'share', 'follow',
    'chapter_read', 'story_complete', 'profile_view', 'link_click',
    'download', 'print', 'copy', 'scroll_90', 'scroll_50'
);

CREATE TYPE engagement_platform AS ENUM ('web', 'ios', 'android', 'api');

CREATE TABLE analytics_engagement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Actor & Target
    actor_id UUID REFERENCES "user"(id) ON DELETE SET NULL,
    story_id UUID REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    author_id UUID REFERENCES author(id) ON DELETE CASCADE,

    -- Engagement Details
    engagement_type engagement_type NOT NULL,
    engagement_value DECIMAL(10, 2), -- For votes: rating value, etc.

    -- Context
    platform engagement_platform DEFAULT 'web',
    session_id UUID,
    referrer_view_id UUID REFERENCES analytics_view(id) ON DELETE SET NULL,

    -- Metadata
    metadata JSONB DEFAULT '{}'::JSONB, -- Flexible additional data

    -- Timestamps
    engaged_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_analytics_engagement_story ON analytics_engagement(story_id) WHERE story_id IS NOT NULL;
CREATE INDEX idx_analytics_engagement_chapter ON analytics_engagement(chapter_id) WHERE chapter_id IS NOT NULL;
CREATE INDEX idx_analytics_engagement_author ON analytics_engagement(author_id) WHERE author_id IS NOT NULL;
CREATE INDEX idx_analytics_engagement_actor ON analytics_engagement(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_analytics_engagement_type ON analytics_engagement(engagement_type);
CREATE INDEX idx_analytics_engagement_date ON analytics_engagement(engaged_at DESC);
CREATE INDEX idx_analytics_engagement_author_date ON analytics_engagement(author_id, engaged_at DESC);
```

#### Table: `analytics_daily_summary`

Pre-aggregated daily analytics for fast dashboard queries.

```sql
CREATE TABLE analytics_daily_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Dimensions
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,
    story_id UUID REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,

    -- Date
    summary_date DATE NOT NULL,

    -- View Metrics
    total_views INTEGER DEFAULT 0,
    unique_views INTEGER DEFAULT 0,
    returning_viewers INTEGER DEFAULT 0,
    average_time_on_page_seconds INTEGER DEFAULT 0,
    bounce_rate DECIMAL(5, 2) DEFAULT 0,

    -- Engagement Metrics
    total_votes INTEGER DEFAULT 0,
    new_favorites INTEGER DEFAULT 0,
    new_bookmarks INTEGER DEFAULT 0,
    total_comments INTEGER DEFAULT 0,
    shares INTEGER DEFAULT 0,

    -- Device Breakdown (JSON for flexibility)
    device_breakdown JSONB DEFAULT '{}'::JSONB, -- {"desktop": 100, "mobile": 200}
    source_breakdown JSONB DEFAULT '{}'::JSONB, -- {"direct": 50, "search": 150}
    country_breakdown JSONB DEFAULT '{}'::JSONB, -- {"US": 100, "UK": 50}

    -- Revenue (if monetization enabled)
    estimated_revenue DECIMAL(10, 2) DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT analytics_daily_summary_unique UNIQUE (author_id, story_id, chapter_id, summary_date),
    CONSTRAINT analytics_daily_summary_date_range CHECK (summary_date <= CURRENT_DATE)
);

-- Indexes
CREATE INDEX idx_analytics_daily_author_date ON analytics_daily_summary(author_id, summary_date DESC);
CREATE INDEX idx_analytics_daily_story_date ON analytics_daily_summary(story_id, summary_date DESC) WHERE story_id IS NOT NULL;
CREATE INDEX idx_analytics_daily_chapter_date ON analytics_daily_summary(chapter_id, summary_date DESC) WHERE chapter_id IS NOT NULL;
```

#### Table: `content_performance`

Long-term performance tracking and trends.

```sql
CREATE TYPE performance_period AS ENUM ('daily', 'weekly', 'monthly', 'quarterly', 'yearly');

CREATE TABLE content_performance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Content Reference
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,

    -- Period
    period_type performance_period NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Aggregate Metrics
    total_views INTEGER DEFAULT 0,
    unique_viewers INTEGER DEFAULT 0,
    total_read_time_hours DECIMAL(10, 2) DEFAULT 0,

    -- Engagement
    vote_count INTEGER DEFAULT 0,
    average_rating DECIMAL(3, 2) DEFAULT 0,
    favorite_count INTEGER DEFAULT 0,
    bookmark_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    share_count INTEGER DEFAULT 0,

    -- Retention
    completion_rate DECIMAL(5, 2) DEFAULT 0, -- % who finish reading
    return_rate DECIMAL(5, 2) DEFAULT 0, -- % who return within 30 days

    -- Growth
    views_growth_rate DECIMAL(5, 2) DEFAULT 0, -- % change from previous period
    engagement_growth_rate DECIMAL(5, 2) DEFAULT 0,

    -- Rankings
    genre_rank INTEGER, -- Rank within genre (if applicable)
    overall_rank INTEGER, -- Overall platform rank

    -- Milestones
    milestones_achieved JSONB DEFAULT '[]'::JSONB, -- ["1k_views", "100_favorites"]

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT content_performance_unique UNIQUE (story_id, chapter_id, period_type, period_start),
    CONSTRAINT content_performance_period_valid CHECK (period_end > period_start),
    CONSTRAINT content_performance_rating_range CHECK (average_rating >= 0 AND average_rating <= 5)
);

-- Indexes
CREATE INDEX idx_content_performance_story ON content_performance(story_id);
CREATE INDEX idx_content_performance_author ON content_performance(author_id);
CREATE INDEX idx_content_performance_period ON content_performance(period_type, period_start DESC);
CREATE INDEX idx_content_performance_rank ON content_performance(genre_rank) WHERE genre_rank IS NOT NULL;
```

### 2.2 API Endpoints

#### Analytics Dashboard Endpoints

| Method | Path                                           | Description                        | Auth Required |
| ------ | ---------------------------------------------- | ---------------------------------- | ------------- |
| GET    | `/api/v1/analytics/dashboard`                  | Get author dashboard overview      | Author        |
| GET    | `/api/v1/analytics/dashboard/stats`            | Get key performance stats          | Author        |
| GET    | `/api/v1/analytics/dashboard/trends`           | Get trend data (views, engagement) | Author        |
| GET    | `/api/v1/analytics/stories`                    | List story analytics               | Author        |
| GET    | `/api/v1/analytics/stories/{storyId}`          | Get detailed story analytics       | Author        |
| GET    | `/api/v1/analytics/stories/{storyId}/chapters` | Get chapter-level analytics        | Author        |
| GET    | `/api/v1/analytics/audience`                   | Get audience demographics          | Author        |
| GET    | `/api/v1/analytics/audience/locations`         | Get geographic data                | Author        |
| GET    | `/api/v1/analytics/audience/devices`           | Get device breakdown               | Author        |
| GET    | `/api/v1/analytics/engagement`                 | Get engagement metrics             | Author        |
| GET    | `/api/v1/analytics/realtime`                   | Get real-time stats                | Author        |

#### Analytics Data Export

| Method | Path                                  | Description                | Auth Required |
| ------ | ------------------------------------- | -------------------------- | ------------- |
| POST   | `/api/v1/analytics/export`            | Request data export        | Author        |
| GET    | `/api/v1/analytics/export/{exportId}` | Get export status/download | Author        |
| GET    | `/api/v1/analytics/export/history`    | List past exports          | Author        |

### 2.3 Request/Response Examples

#### Get Dashboard Overview

**Request:**

```json
GET /api/v1/analytics/dashboard?period=30d
```

**Response:**

```json
{
    "data": {
        "period": {
            "start": "2025-12-15",
            "end": "2026-01-14",
            "days": 30
        },
        "overview": {
            "totalViews": 15420,
            "uniqueReaders": 8750,
            "totalReadingTime": "142 hours",
            "averageRating": 4.3,
            "newFollowers": 125
        },
        "trends": {
            "views": {
                "current": 15420,
                "previous": 12300,
                "changePercent": 25.4
            },
            "engagement": {
                "current": 8.5,
                "previous": 7.2,
                "changePercent": 18.1
            }
        },
        "topStories": [
            {
                "storyId": "uuid",
                "title": "My Best Story",
                "views": 5200,
                "engagementRate": 12.3
            }
        ],
        "demographics": {
            "devices": {
                "mobile": 65,
                "desktop": 30,
                "tablet": 5
            },
            "topCountries": [
                { "code": "US", "name": "United States", "percent": 45 },
                { "code": "GB", "name": "United Kingdom", "percent": 15 }
            ]
        }
    }
}
```

#### Get Story Analytics

**Request:**

```json
GET /api/v1/analytics/stories/{storyId}?granularity=daily&startDate=2025-12-01&endDate=2026-01-14
```

**Response:**

```json
{
    "data": {
        "story": {
            "id": "uuid",
            "title": "My Great Story",
            "totalViews": 15420,
            "uniqueReaders": 8750,
            "averageRating": 4.3,
            "totalFavorites": 450,
            "totalBookmarks": 320
        },
        "dailyData": [
            {
                "date": "2026-01-14",
                "views": 520,
                "uniqueViews": 410,
                "engagement": {
                    "votes": 12,
                    "comments": 5,
                    "favorites": 8
                }
            }
        ],
        "chapterPerformance": [
            {
                "chapterId": "uuid",
                "number": 1,
                "title": "Chapter 1",
                "views": 15420,
                "completionRate": 85.5,
                "averageTime": "12 minutes"
            }
        ],
        "trafficSources": [
            { "source": "direct", "views": 8000, "percent": 51.9 },
            { "source": "search", "views": 4500, "percent": 29.2 },
            { "source": "social", "views": 1800, "percent": 11.7 }
        ]
    }
}
```

### 2.4 Integration Points

- **View Tracking**: Middleware intercepts story/chapter reads to populate `analytics_view`
- **Engagement Sync**: Triggers on vote, favorite, bookmark, comment actions populate `analytics_engagement`
- **Daily Aggregation**: Scheduled job aggregates data into `analytics_daily_summary`
- **Real-time Updates**: WebSocket connections push live stats to author dashboard
- **Performance Scoring**: Algorithm calculates rankings and trends in `content_performance`

---

## 3. Monetization Features

### 3.1 Database Schema

#### Table: `subscription_plan`

Available subscription tiers.

```sql
CREATE TYPE subscription_interval AS ENUM ('monthly', 'quarterly', 'yearly');
CREATE TYPE subscription_currency AS ENUM ('USD', 'EUR', 'GBP', 'JPY', 'CAD', 'AUD');

CREATE TABLE subscription_plan (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Plan Information
    name CHAR(100) NOT NULL,
    slug CHAR(100) NOT NULL UNIQUE,
    description TEXT,

    -- Pricing
    price_amount DECIMAL(10, 2) NOT NULL,
    currency subscription_currency DEFAULT 'USD',
    interval subscription_interval NOT NULL,

    -- Features
    features JSONB DEFAULT '{}'::JSONB, -- {"unlimited_stories": true, "offline_access": true}
    max_stories INTEGER, -- NULL = unlimited
    max_offline_downloads INTEGER,
    analytics_level CHAR(20), -- basic, advanced, premium
    support_level CHAR(20), -- community, email, priority

    -- Visibility
    is_active BOOLEAN DEFAULT TRUE,
    display_order INTEGER DEFAULT 0,
    is_recommended BOOLEAN DEFAULT FALSE,

    -- Trial
    trial_days INTEGER DEFAULT 0,
    trial_features JSONB DEFAULT '{}'::JSONB,

    -- Limits
    max_word_count_per_story INTEGER,
    allow_monetization BOOLEAN DEFAULT FALSE,
    allow_collaboration BOOLEAN DEFAULT FALSE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_subscription_plans_active ON subscription_plan(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscription_plans_order ON subscription_plan(display_order) WHERE is_active = TRUE AND deleted_at IS NULL;
```

#### Table: `user_subscription`

User subscription records.

```sql
CREATE TYPE subscription_status AS ENUM ('trialing', 'active', 'past_due', 'canceled', 'unpaid', 'paused');

CREATE TABLE user_subscription (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plan(id) ON DELETE CASCADE,

    -- Subscription Status
    status subscription_status DEFAULT 'trialing',

    -- Billing
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT FALSE,
    canceled_at TIMESTAMP WITH TIME ZONE,

    -- Payment Provider
    provider_subscription_id CHAR(255), -- Stripe/PayPal subscription ID
    provider_customer_id CHAR(255),

    -- Trial
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT user_subscriptions_user_unique UNIQUE (user_id)
);

-- Indexes
CREATE INDEX idx_user_subscriptions_user ON user_subscription(user_id);
CREATE INDEX idx_user_subscriptions_plan ON user_subscription(plan_id);
CREATE INDEX idx_user_subscriptions_status ON user_subscription(status);
CREATE INDEX idx_user_subscriptions_period_end ON user_subscription(current_period_end);
CREATE INDEX idx_user_subscriptions_provider ON user_subscription(provider_subscription_id) WHERE provider_subscription_id IS NOT NULL;
```

#### Table: `payment_transaction`

Payment history and transactions.

```sql
CREATE TYPE payment_status AS ENUM ('pending', 'processing', 'succeeded', 'failed', 'refunded', 'disputed');
CREATE TYPE payment_method_type AS ENUM ('card', 'paypal', 'bank_transfer', 'crypto', 'credits');

CREATE TABLE payment_transaction (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES user_subscription(id) ON DELETE SET NULL,

    -- Transaction Details
    amount DECIMAL(10, 2) NOT NULL,
    currency subscription_currency DEFAULT 'USD',
    status payment_status DEFAULT 'pending',
    payment_method payment_method_type NOT NULL,

    -- Provider Info
    provider_transaction_id CHAR(255),
    provider_payment_intent_id CHAR(255),

    -- Description
    description TEXT,
    invoice_url TEXT,
    receipt_url TEXT,

    -- Refund Info
    refund_amount DECIMAL(10, 2) DEFAULT 0,
    refund_reason TEXT,
    refunded_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}'::JSONB,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT payment_transactions_amount_positive CHECK (amount > 0)
);

-- Indexes
CREATE INDEX idx_payment_transactions_user ON payment_transaction(user_id);
CREATE INDEX idx_payment_transactions_subscription ON payment_transaction(subscription_id) WHERE subscription_id IS NOT NULL;
CREATE INDEX idx_payment_transactions_status ON payment_transaction(status);
CREATE INDEX idx_payment_transactions_date ON payment_transaction(created_at DESC);
CREATE INDEX idx_payment_transactions_provider ON payment_transaction(provider_transaction_id) WHERE provider_transaction_id IS NOT NULL;
```

#### Table: `premium_content`

Paywalled/premium story content.

```sql
CREATE TYPE premium_access_type AS ENUM ('subscription', 'one_time', 'free_preview', 'ad_supported');

CREATE TABLE premium_content (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Content Reference
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE, -- NULL = entire story is premium

    -- Access Configuration
    access_type premium_access_type DEFAULT 'subscription',
    required_plan_id UUID REFERENCES subscription_plan(id) ON DELETE SET NULL,

    -- One-time Purchase
    price_amount DECIMAL(10, 2),
    currency subscription_currency DEFAULT 'USD',

    -- Preview Configuration
    preview_percentage INTEGER DEFAULT 20, -- % free to read
    preview_word_count INTEGER, -- Alternative: X words free
    preview_paragraphs INTEGER,

    -- Access Rules
    early_access_days INTEGER DEFAULT 0, -- Early access for premium users
    subscriber_discount_percent DECIMAL(5, 2) DEFAULT 0,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    starts_at TIMESTAMP WITH TIME ZONE,
    ends_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT premium_content_percentage_range CHECK (preview_percentage >= 0 AND preview_percentage <= 100),
    CONSTRAINT premium_content_discount_range CHECK (subscriber_discount_percent >= 0 AND subscriber_discount_percent <= 100)
);

-- Indexes
CREATE INDEX idx_premium_content_story ON premium_content(story_id);
CREATE INDEX idx_premium_content_chapter ON premium_content(chapter_id) WHERE chapter_id IS NOT NULL;
CREATE INDEX idx_premium_content_active ON premium_content(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_premium_content_access_type ON premium_content(access_type);
```

#### Table: `content_purchase`

Individual content purchases.

```sql
CREATE TABLE content_purchase (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    premium_content_id UUID NOT NULL REFERENCES premium_content(id) ON DELETE CASCADE,
    transaction_id UUID REFERENCES payment_transaction(id) ON DELETE SET NULL,

    -- Purchase Details
    price_paid DECIMAL(10, 2) NOT NULL,
    currency subscription_currency DEFAULT 'USD',

    -- Access
    has_access BOOLEAN DEFAULT TRUE,
    access_granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    access_expires_at TIMESTAMP WITH TIME ZONE, -- NULL = permanent

    -- Refund
    is_refunded BOOLEAN DEFAULT FALSE,
    refunded_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT content_purchases_user_content_unique UNIQUE (user_id, premium_content_id)
);

-- Indexes
CREATE INDEX idx_content_purchases_user ON content_purchase(user_id);
CREATE INDEX idx_content_purchases_content ON content_purchase(premium_content_id);
CREATE INDEX idx_content_purchases_active ON content_purchase(user_id, has_access) WHERE has_access = TRUE;
```

#### Table: `revenue_share`

Author revenue tracking and payouts.

```sql
CREATE TYPE payout_status AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE revenue_share (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Author & Content
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,
    story_id UUID REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,

    -- Revenue Source
    source_type CHAR(50) NOT NULL, -- subscription_share, purchase, tip, bonus
    source_id UUID, -- Reference to subscription, purchase, etc.

    -- Revenue Amounts
    gross_amount DECIMAL(10, 2) NOT NULL,
    platform_fee DECIMAL(10, 2) NOT NULL,
    net_amount DECIMAL(10, 2) NOT NULL,
    currency subscription_currency DEFAULT 'USD',

    -- Share Calculation
    revenue_share_percent DECIMAL(5, 2) DEFAULT 70.00, -- 70% to author

    -- Status
    is_paid_out BOOLEAN DEFAULT FALSE,
    payout_id UUID, -- Reference to payout batch
    paid_out_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT revenue_share_percent_range CHECK (revenue_share_percent >= 0 AND revenue_share_percent <= 100),
    CONSTRAINT revenue_share_calculation CHECK (net_amount = gross_amount - platform_fee)
);

-- Indexes
CREATE INDEX idx_revenue_share_author ON revenue_share(author_id);
CREATE INDEX idx_revenue_share_story ON revenue_share(story_id) WHERE story_id IS NOT NULL;
CREATE INDEX idx_revenue_share_unpaid ON revenue_share(author_id, is_paid_out) WHERE is_paid_out = FALSE;
CREATE INDEX idx_revenue_share_payout ON revenue_share(payout_id) WHERE payout_id IS NOT NULL;
CREATE INDEX idx_revenue_share_created ON revenue_share(created_at DESC);
```

#### Table: `author_payout`

Author payout batches.

```sql
CREATE TABLE author_payout (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Author
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,

    -- Payout Details
    total_amount DECIMAL(10, 2) NOT NULL,
    currency subscription_currency DEFAULT 'USD',
    status payout_status DEFAULT 'pending',

    -- Payment Method
    payment_method payment_method_type NOT NULL,
    payment_details JSONB DEFAULT '{}'::JSONB, -- Encrypted payment info

    -- Provider
    provider_payout_id CHAR(255), -- Stripe/PayPal payout ID

    -- Period
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Processing
    processed_at TIMESTAMP WITH TIME ZONE,
    processed_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    failure_reason TEXT,

    -- Receipt
    receipt_url TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_author_payouts_author ON author_payout(author_id);
CREATE INDEX idx_author_payouts_status ON author_payout(status);
CREATE INDEX idx_author_payouts_period ON author_payout(period_start, period_end);
```

### 3.2 API Endpoints

#### Subscription Management

| Method | Path                                       | Description              | Auth Required |
| ------ | ------------------------------------------ | ------------------------ | ------------- |
| GET    | `/api/v1/billing/plans`                    | List available plans     | No            |
| GET    | `/api/v1/billing/plans/{planId}`           | Get plan details         | No            |
| POST   | `/api/v1/billing/subscribe`                | Subscribe to plan        | User          |
| GET    | `/api/v1/billing/subscription`             | Get current subscription | User          |
| PUT    | `/api/v1/billing/subscription`             | Update subscription      | User          |
| DELETE | `/api/v1/billing/subscription`             | Cancel subscription      | User          |
| POST   | `/api/v1/billing/subscription/reactivate`  | Reactivate subscription  | User          |
| POST   | `/api/v1/billing/subscription/change-plan` | Change subscription plan | User          |

#### Payment & Billing

| Method | Path                                                 | Description                | Auth Required |
| ------ | ---------------------------------------------------- | -------------------------- | ------------- |
| GET    | `/api/v1/billing/payments`                           | List payment history       | User          |
| GET    | `/api/v1/billing/payments/{paymentId}`               | Get payment details        | User          |
| GET    | `/api/v1/billing/invoices`                           | List invoices              | User          |
| GET    | `/api/v1/billing/invoices/{invoiceId}/download`      | Download invoice           | User          |
| POST   | `/api/v1/billing/payment-methods`                    | Add payment method         | User          |
| GET    | `/api/v1/billing/payment-methods`                    | List payment methods       | User          |
| DELETE | `/api/v1/billing/payment-methods/{methodId}`         | Remove payment method      | User          |
| PUT    | `/api/v1/billing/payment-methods/{methodId}/default` | Set default payment method | User          |

#### Premium Content

| Method | Path                                                   | Description                 | Auth Required |
| ------ | ------------------------------------------------------ | --------------------------- | ------------- |
| GET    | `/api/v1/billing/premium-content`                      | List premium content        | No            |
| GET    | `/api/v1/billing/premium-content/{contentId}`          | Get premium content details | No            |
| POST   | `/api/v1/billing/premium-content/{contentId}/purchase` | Purchase content            | User          |
| GET    | `/api/v1/billing/purchases`                            | List purchased content      | User          |
| GET    | `/api/v1/billing/purchases/{purchaseId}`               | Get purchase details        | User          |

#### Author Revenue

| Method | Path                                        | Description               | Auth Required |
| ------ | ------------------------------------------- | ------------------------- | ------------- |
| GET    | `/api/v1/billing/revenue`                   | Get revenue summary       | Author        |
| GET    | `/api/v1/billing/revenue/transactions`      | List revenue transactions | Author        |
| GET    | `/api/v1/billing/revenue/payouts`           | List payout history       | Author        |
| GET    | `/api/v1/billing/revenue/payouts/scheduled` | Get upcoming payout       | Author        |
| PUT    | `/api/v1/billing/revenue/payout-settings`   | Update payout settings    | Author        |
| POST   | `/api/v1/billing/revenue/request-payout`    | Request early payout      | Author        |

### 3.3 Request/Response Examples

#### Subscribe to Plan

**Request:**

```json
POST /api/v1/billing/subscribe
{
  "planId": "uuid",
  "paymentMethodId": "pm_1234567890",
  "couponCode": "WELCOME20",
  "billingAddress": {
    "country": "US",
    "postalCode": "12345"
  }
}
```

**Response:**

```json
{
    "data": {
        "subscription": {
            "id": "uuid",
            "plan": {
                "id": "uuid",
                "name": "Pro Author",
                "price": 19.99,
                "currency": "USD",
                "interval": "monthly"
            },
            "status": "active",
            "currentPeriodStart": "2026-01-14T10:30:00Z",
            "currentPeriodEnd": "2026-02-14T10:30:00Z",
            "trialEnd": null,
            "cancelAtPeriodEnd": false
        },
        "payment": {
            "id": "uuid",
            "amount": 15.99,
            "currency": "USD",
            "status": "succeeded",
            "discountApplied": 4.0
        }
    },
    "message": "Subscription activated successfully"
}
```

#### Get Revenue Summary

**Request:**

```json
GET /api/v1/billing/revenue?period=30d
```

**Response:**

```json
{
    "data": {
        "period": {
            "start": "2025-12-15",
            "end": "2026-01-14"
        },
        "summary": {
            "totalRevenue": 1250.5,
            "platformFees": 375.15,
            "netEarnings": 875.35,
            "paidOut": 500.0,
            "pending": 375.35
        },
        "bySource": {
            "subscriptionShare": 800.0,
            "contentPurchases": 350.5,
            "tips": 100.0
        },
        "byStory": [
            {
                "storyId": "uuid",
                "title": "My Best Story",
                "revenue": 850.25
            }
        ],
        "upcomingPayout": {
            "date": "2026-02-01",
            "estimatedAmount": 375.35
        }
    }
}
```

### 3.4 Integration Points

- **Payment Webhooks**: Stripe/PayPal webhooks update subscription/payment status
- **Access Control**: Middleware checks `user_subscription` and `content_purchase` for content access
- **Revenue Calculation**: Nightly job calculates author revenue shares
- **Analytics Integration**: Revenue data feeds into `analytics_daily_summary.estimated_revenue`
- **Tax Handling**: Integration with tax calculation services based on user location

---

## 4. Collaboration Features

### 4.1 Database Schema

#### Table: `workspace`

Collaborative writing workspaces.

```sql
CREATE TYPE workspace_role AS ENUM ('owner', 'admin', 'editor', 'reviewer', 'viewer');
CREATE TYPE workspace_status AS ENUM ('active', 'archived', 'suspended');

CREATE TABLE workspace (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Workspace Info
    name CHAR(255) NOT NULL,
    description TEXT,
    slug CHAR(255) NOT NULL UNIQUE,

    -- Owner
    owner_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Settings
    is_private BOOLEAN DEFAULT TRUE,
    allow_guest_access BOOLEAN DEFAULT FALSE,
    max_members INTEGER DEFAULT 10,

    -- Status
    status workspace_status DEFAULT 'active',

    -- Branding
    cover_image_id UUID REFERENCES file(id) ON DELETE SET NULL,
    icon_image_id UUID REFERENCES file(id) ON DELETE SET NULL,

    -- Statistics
    total_members INTEGER DEFAULT 1,
    total_stories INTEGER DEFAULT 0,
    total_documents INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_workspaces_owner ON workspace(owner_id);
CREATE INDEX idx_workspaces_slug ON workspace(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspaces_status ON workspace(status) WHERE deleted_at IS NULL;
```

#### Table: `workspace_member`

Members within a workspace.

```sql
CREATE TABLE workspace_member (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    workspace_id UUID NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    invited_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Role & Permissions
    role workspace_role DEFAULT 'viewer',
    permissions JSONB DEFAULT '{}'::JSONB, -- Custom permissions override

    -- Invitation
    invitation_token CHAR(255),
    invitation_expires_at TIMESTAMP WITH TIME ZONE,
    is_invitation_accepted BOOLEAN DEFAULT FALSE,
    joined_at TIMESTAMP WITH TIME ZONE,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_active_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT workspace_members_unique UNIQUE (workspace_id, user_id)
);

-- Indexes
CREATE INDEX idx_workspace_members_workspace ON workspace_member(workspace_id);
CREATE INDEX idx_workspace_members_user ON workspace_member(user_id);
CREATE INDEX idx_workspace_members_role ON workspace_member(workspace_id, role);
CREATE INDEX idx_workspace_members_active ON workspace_member(workspace_id, is_active) WHERE is_active = TRUE;
```

#### Table: `co_author`

Co-author relationships for stories.

```sql
CREATE TYPE co_author_role AS ENUM ('primary', 'co_author', 'contributor', 'editor');
CREATE TYPE co_author_permission AS ENUM ('read', 'write', 'publish', 'admin');

CREATE TABLE co_author (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES author(id) ON DELETE CASCADE,
    invited_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Role
    role co_author_role DEFAULT 'contributor',

    -- Permissions
    can_edit BOOLEAN DEFAULT FALSE,
    can_publish BOOLEAN DEFAULT FALSE,
    can_invite BOOLEAN DEFAULT FALSE,
    permissions co_author_permission[] DEFAULT ARRAY['read']::co_author_permission[],

    -- Revenue Share
    revenue_share_percent DECIMAL(5, 2) DEFAULT 0,

    -- Attribution
    display_order INTEGER DEFAULT 0,
    is_visible BOOLEAN DEFAULT TRUE, -- Show on story page

    -- Status
    status CHAR(20) DEFAULT 'pending', -- pending, active, removed

    -- Invitation
    invitation_message TEXT,
    responded_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT co_authors_story_author_unique UNIQUE (story_id, author_id),
    CONSTRAINT co_authors_revenue_share_range CHECK (revenue_share_percent >= 0 AND revenue_share_percent <= 100)
);

-- Indexes
CREATE INDEX idx_co_authors_story ON co_author(story_id);
CREATE INDEX idx_co_authors_author ON co_author(author_id);
CREATE INDEX idx_co_authors_active ON co_author(story_id, status) WHERE status = 'active';
```

#### Table: `editorial_review`

Editorial review workflow.

```sql
CREATE TYPE review_status AS ENUM ('pending', 'in_review', 'approved', 'rejected', 'needs_revision');
CREATE TYPE review_priority AS ENUM ('low', 'medium', 'high', 'urgent');

CREATE TABLE editorial_review (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    requester_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    reviewer_id UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Review Details
    title CHAR(255) NOT NULL,
    description TEXT,
    priority review_priority DEFAULT 'medium',

    -- Status
    status review_status DEFAULT 'pending',

    -- Content Version
    document_version_id UUID REFERENCES document_version(id) ON DELETE SET NULL,

    -- Review Content
    reviewer_notes TEXT,
    suggestions JSONB DEFAULT '[]'::JSONB, -- Structured suggestions

    -- Approval
    approved_at TIMESTAMP WITH TIME ZONE,
    rejected_at TIMESTAMP WITH TIME ZONE,
    revision_requested_at TIMESTAMP WITH TIME ZONE,

    -- Due Date
    due_date TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_editorial_reviews_story ON editorial_review(story_id);
CREATE INDEX idx_editorial_reviews_chapter ON editorial_review(chapter_id) WHERE chapter_id IS NOT NULL;
CREATE INDEX idx_editorial_reviews_requester ON editorial_review(requester_id);
CREATE INDEX idx_editorial_reviews_reviewer ON editorial_review(reviewer_id) WHERE reviewer_id IS NOT NULL;
CREATE INDEX idx_editorial_reviews_status ON editorial_review(status);
CREATE INDEX idx_editorial_reviews_pending ON editorial_review(reviewer_id, status) WHERE status IN ('pending', 'in_review');
CREATE INDEX idx_editorial_reviews_due_date ON editorial_review(due_date) WHERE status IN ('pending', 'in_review');
```

#### Table: `editorial_comment`

Inline comments on editorial reviews.

```sql
CREATE TABLE editorial_comment (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    review_id UUID NOT NULL REFERENCES editorial_review(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES editorial_comment(id) ON DELETE CASCADE,

    -- Content
    content TEXT NOT NULL,

    -- Position (for inline comments)
    position_start INTEGER,
    position_end INTEGER,
    paragraph_index INTEGER,

    -- Status
    is_resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_editorial_comments_review ON editorial_comment(review_id);
CREATE INDEX idx_editorial_comments_author ON editorial_comment(author_id);
CREATE INDEX idx_editorial_comments_unresolved ON editorial_comment(review_id, is_resolved) WHERE is_resolved = FALSE;
```

#### Table: `writing_session`

Real-time collaborative writing sessions.

```sql
CREATE TABLE writing_session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    story_id UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    chapter_id UUID REFERENCES chapter(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES document(id) ON DELETE CASCADE,

    -- Session Info
    name CHAR(255),
    created_by UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP WITH TIME ZONE,

    -- Participants
    participant_count INTEGER DEFAULT 1,
    max_participants INTEGER DEFAULT 5,

    -- Locking
    is_locked BOOLEAN DEFAULT FALSE,
    locked_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    locked_at TIMESTAMP WITH TIME ZONE,

    -- Version Control
    base_version_id UUID REFERENCES document_version(id) ON DELETE SET NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_writing_sessions_story ON writing_session(story_id);
CREATE INDEX idx_writing_sessions_active ON writing_session(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_writing_sessions_created_by ON writing_session(created_by);
```

#### Table: `writing_session_participant`

Participants in writing sessions.

```sql
CREATE TABLE writing_session_participant (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- References
    session_id UUID NOT NULL REFERENCES writing_session(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Role
    is_host BOOLEAN DEFAULT FALSE,
    can_edit BOOLEAN DEFAULT TRUE,

    -- Status
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP WITH TIME ZONE,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Cursor/Selection
    cursor_position JSONB, -- {"paragraph": 5, "offset": 120}
    selection_range JSONB, -- {"start": {"p": 5, "o": 10}, "end": {"p": 5, "o": 50}}

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT writing_session_participants_unique UNIQUE (session_id, user_id)
);

-- Indexes
CREATE INDEX idx_writing_session_participants_session ON writing_session_participant(session_id);
CREATE INDEX idx_writing_session_participants_user ON writing_session_participant(user_id);
CREATE INDEX idx_writing_session_participants_active ON writing_session_participant(session_id, left_at) WHERE left_at IS NULL;
```

### 4.2 API Endpoints

#### Workspace Management

| Method | Path                                                     | Description            | Auth Required    |
| ------ | -------------------------------------------------------- | ---------------------- | ---------------- |
| GET    | `/api/v1/workspaces`                                     | List user's workspaces | User             |
| POST   | `/api/v1/workspaces`                                     | Create new workspace   | User             |
| GET    | `/api/v1/workspaces/{workspaceId}`                       | Get workspace details  | Workspace Member |
| PUT    | `/api/v1/workspaces/{workspaceId}`                       | Update workspace       | Workspace Admin  |
| DELETE | `/api/v1/workspaces/{workspaceId}`                       | Delete workspace       | Workspace Owner  |
| POST   | `/api/v1/workspaces/{workspaceId}/invite`                | Invite member          | Workspace Admin  |
| POST   | `/api/v1/workspaces/{workspaceId}/invite/accept`         | Accept invitation      | User             |
| DELETE | `/api/v1/workspaces/{workspaceId}/members/{userId}`      | Remove member          | Workspace Admin  |
| PUT    | `/api/v1/workspaces/{workspaceId}/members/{userId}/role` | Update member role     | Workspace Admin  |
| GET    | `/api/v1/workspaces/{workspaceId}/members`               | List members           | Workspace Member |
| GET    | `/api/v1/workspaces/{workspaceId}/activity`              | Get workspace activity | Workspace Member |

#### Co-Author Management

| Method | Path                                                  | Description              | Auth Required |
| ------ | ----------------------------------------------------- | ------------------------ | ------------- |
| GET    | `/api/v1/stories/{storyId}/co-authors`                | List co-authors          | Story Access  |
| POST   | `/api/v1/stories/{storyId}/co-authors`                | Add co-author            | Story Owner   |
| PUT    | `/api/v1/stories/{storyId}/co-authors/{authorId}`     | Update co-author         | Story Owner   |
| DELETE | `/api/v1/stories/{storyId}/co-authors/{authorId}`     | Remove co-author         | Story Owner   |
| POST   | `/api/v1/stories/{storyId}/co-authors/invite/accept`  | Accept co-author invite  | User          |
| POST   | `/api/v1/stories/{storyId}/co-authors/invite/decline` | Decline co-author invite | User          |

#### Editorial Review

| Method | Path                                  | Description              | Auth Required      |
| ------ | ------------------------------------- | ------------------------ | ------------------ |
| POST   | `/api/v1/stories/{storyId}/reviews`   | Request editorial review | Story Owner        |
| GET    | `/api/v1/stories/{storyId}/reviews`   | List reviews             | Story Access       |
| GET    | `/api/v1/reviews/{reviewId}`          | Get review details       | Review Participant |
| PUT    | `/api/v1/reviews/{reviewId}`          | Update review            | Reviewer           |
| POST   | `/api/v1/reviews/{reviewId}/comments` | Add review comment       | Review Participant |
| GET    | `/api/v1/reviews/{reviewId}/comments` | List review comments     | Review Participant |
| PUT    | `/api/v1/reviews/{reviewId}/status`   | Update review status     | Reviewer           |
| POST   | `/api/v1/reviews/{reviewId}/resolve`  | Resolve review           | Requester          |
| GET    | `/api/v1/reviews/assigned`            | Get my assigned reviews  | User               |

#### Writing Sessions

| Method | Path                                        | Description            | Auth Required       |
| ------ | ------------------------------------------- | ---------------------- | ------------------- |
| POST   | `/api/v1/documents/{documentId}/sessions`   | Start writing session  | Document Access     |
| GET    | `/api/v1/sessions/{sessionId}`              | Get session details    | Session Participant |
| POST   | `/api/v1/sessions/{sessionId}/join`         | Join session           | Document Access     |
| POST   | `/api/v1/sessions/{sessionId}/leave`        | Leave session          | Session Participant |
| POST   | `/api/v1/sessions/{sessionId}/end`          | End session            | Session Host        |
| GET    | `/api/v1/sessions/{sessionId}/participants` | List participants      | Session Participant |
| PUT    | `/api/v1/sessions/{sessionId}/cursor`       | Update cursor position | Session Participant |
| POST   | `/api/v1/sessions/{sessionId}/lock`         | Lock document          | Session Host        |
| DELETE | `/api/v1/sessions/{sessionId}/lock`         | Unlock document        | Session Host        |
| POST   | `/api/v1/sessions/{sessionId}/save`         | Save session changes   | Session Host        |

### 4.3 Request/Response Examples

#### Create Workspace

**Request:**

```json
POST /api/v1/workspaces
{
  "name": "Fantasy Writers Collective",
  "description": "A collaborative workspace for fantasy authors",
  "isPrivate": true,
  "maxMembers": 15
}
```

**Response:**

```json
{
    "data": {
        "id": "uuid",
        "name": "Fantasy Writers Collective",
        "slug": "fantasy-writers-collective",
        "description": "A collaborative workspace for fantasy authors",
        "ownerId": "uuid",
        "isPrivate": true,
        "totalMembers": 1,
        "totalStories": 0,
        "createdAt": "2026-01-14T10:30:00Z",
        "inviteLink": "https://samsa.io/w/fantasy-writers-collective/invite/abc123"
    }
}
```

#### Add Co-Author

**Request:**

```json
POST /api/v1/stories/{storyId}/co-authors
{
  "authorId": "uuid",
  "role": "co_author",
  "permissions": ["read", "write", "publish"],
  "revenueSharePercent": 30,
  "invitationMessage": "Would you like to collaborate on this story?"
}
```

**Response:**

```json
{
    "data": {
        "id": "uuid",
        "author": {
            "id": "uuid",
            "stageName": "Jane Smith",
            "slug": "jane-smith"
        },
        "role": "co_author",
        "permissions": ["read", "write", "publish"],
        "revenueSharePercent": 30,
        "status": "pending",
        "invitedAt": "2026-01-14T10:30:00Z",
        "message": "Invitation sent successfully"
    }
}
```

#### Request Editorial Review

**Request:**

```json
POST /api/v1/stories/{storyId}/reviews
{
  "chapterId": "uuid",
  "title": "Grammar and Flow Review",
  "description": "Please review Chapter 3 for grammar issues and flow improvements",
  "priority": "medium",
  "dueDate": "2026-01-21T10:30:00Z",
  "reviewerId": "uuid"
}
```

**Response:**

```json
{
    "data": {
        "id": "uuid",
        "storyId": "uuid",
        "chapterId": "uuid",
        "title": "Grammar and Flow Review",
        "status": "pending",
        "priority": "medium",
        "dueDate": "2026-01-21T10:30:00Z",
        "reviewer": {
            "id": "uuid",
            "stageName": "Editor Pro"
        },
        "createdAt": "2026-01-14T10:30:00Z"
    }
}
```

### 4.4 Integration Points

- **Real-time Collaboration**: WebSocket connections for live writing sessions
- **Version Control**: Document versions created on session save
- **Notification System**: Triggers for invitations, review assignments, comments
- **Access Control**: Middleware checks workspace/co-author permissions
- **Activity Tracking**: Workspace actions logged to `activity` table
- **Revenue Sharing**: Co-author revenue splits integrated with `revenue_share`

---

## 5. Discovery Features

### 5.1 Database Schema

#### Table: `user_preference`

User content preferences for personalization.

```sql
CREATE TABLE user_preference (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- User Reference
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Genre Preferences
    preferred_genres UUID[] DEFAULT '{}'::UUID[], -- References genre.id
    disliked_genres UUID[] DEFAULT '{}'::UUID[],

    -- Tag Preferences
    preferred_tags UUID[] DEFAULT '{}'::UUID[],
    disliked_tags UUID[] DEFAULT '{}'::UUID[],

    -- Content Preferences
    preferred_story_length CHAR(20), -- short, medium, long, any
    preferred_update_frequency CHAR(20), -- daily, weekly, monthly, any
    preferred_content_rating CHAR(20), -- all, mature, teen, general

    -- Language
    preferred_languages CHAR(3)[] DEFAULT ARRAY['en']::CHAR(3)[],

    -- Notification Preferences
    notify_new_chapters BOOLEAN DEFAULT TRUE,
    notify_new_followers BOOLEAN DEFAULT TRUE,
    notify_recommendations BOOLEAN DEFAULT TRUE,
    notify_daily_digest BOOLEAN DEFAULT FALSE,

    -- Discovery Settings
    show_mature_content BOOLEAN DEFAULT FALSE,
    show_completed_only BOOLEAN DEFAULT FALSE,
    hide_read_stories BOOLEAN DEFAULT TRUE,

    -- Algorithm Settings
    recommendation_diversity DECIMAL(3, 2) DEFAULT 0.3, -- 0-1, higher = more diverse

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT user_preferences_user_unique UNIQUE (user_id),
    CONSTRAINT user_preferences_diversity_range CHECK (recommendation_diversity >= 0 AND recommendation_diversity <= 1)
);

-- Indexes
CREATE INDEX idx_user_preferences_user ON user_preference(user_id);
```

#### Table: `recommendation`

Generated content recommendations.

```sql
CREATE TYPE recommendation_type AS ENUM ('trending', 'similar', 'collaborative', 'editor_pick', 'genre_match', 'author_follow', 'new_release', 'continue_reading');
CREATE TYPE recommendation_reason AS ENUM ('because_you_read', 'trending_now', 'similar_to_favorites', 'authors_you_follow', 'editor_pick', 'popular_in_genre', 'new_from_favorite_author', 'based_on_preferences');

CREATE TABLE recommendation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Target User
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Recommended Content
    story_id UUID REFERENCES story(id) ON DELETE CASCADE,
    author_id UUID REFERENCES author(id) ON DELETE CASCADE,

    -- Recommendation Type
    type recommendation_type NOT NULL,
    reason recommendation_reason NOT NULL,

    -- Scoring
    confidence_score DECIMAL(5, 4) NOT NULL, -- 0-1, higher = more confident
    rank INTEGER NOT NULL,

    -- Context (for explainability)
    context JSONB DEFAULT '{}'::JSONB, -- {"similar_story_id": "uuid", "shared_genres": ["fantasy"]}

    -- Status
    is_seen BOOLEAN DEFAULT FALSE,
    seen_at TIMESTAMP WITH TIME ZONE,
    is_clicked BOOLEAN DEFAULT FALSE,
    clicked_at TIMESTAMP WITH TIME ZONE,

    -- Interaction
    is_dismissed BOOLEAN DEFAULT FALSE,
    dismissed_at TIMESTAMP WITH TIME ZONE,
    dismissal_reason CHAR(50), -- not_interested, already_read, etc.

    -- Timestamps
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_recommendations_user ON recommendation(user_id);
CREATE INDEX idx_recommendations_user_type ON recommendation(user_id, type);
CREATE INDEX idx_recommendations_user_unseen ON recommendation(user_id, is_seen, rank) WHERE is_seen = FALSE;
CREATE INDEX idx_recommendations_story ON recommendation(story_id) WHERE story_id IS NOT NULL;
CREATE INDEX idx_recommendations_generated ON recommendation(generated_at DESC);
CREATE INDEX idx_recommendations_expires ON recommendation(expires_at) WHERE expires_at IS NOT NULL;
```

#### Table: `content_similarity`

Pre-calculated content similarity scores.

```sql
CREATE TABLE content_similarity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Content Pair
    story_id_1 UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,
    story_id_2 UUID NOT NULL REFERENCES story(id) ON DELETE CASCADE,

    -- Similarity Scores (0-1)
    overall_similarity DECIMAL(5, 4) NOT NULL,
    genre_similarity DECIMAL(5, 4) DEFAULT 0,
    tag_similarity DECIMAL(5, 4) DEFAULT 0,
    theme_similarity DECIMAL(5, 4) DEFAULT 0,
    style_similarity DECIMAL(5, 4) DEFAULT 0,

    -- Metadata
    common_genres UUID[] DEFAULT '{}'::UUID[],
    common_tags UUID[] DEFAULT '{}'::UUID[],

    -- Calculation Info
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    algorithm_version CHAR(20) DEFAULT '1.0',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT content_similarity_pair_unique UNIQUE (story_id_1, story_id_2),
    CONSTRAINT content_similarity_different_stories CHECK (story_id_1 <> story_id_2),
    CONSTRAINT content_similarity_overall_range CHECK (overall_similarity >= 0 AND overall_similarity <= 1)
);

-- Indexes
CREATE INDEX idx_content_similarity_story1 ON content_similarity(story_id_1);
CREATE INDEX idx_content_similarity_story2 ON content_similarity(story_id_2);
CREATE INDEX idx_content_similarity_score ON content_similarity(story_id_1, overall_similarity DESC);
CREATE INDEX idx_content_similarity_high ON content_similarity(story_id_1, overall_similarity DESC) WHERE overall_similarity >= 0.7;
```

#### Table: `discovery_feed`

Personalized discovery feed.

```sql
CREATE TYPE feed_item_type AS ENUM ('story', 'author', 'collection', 'genre_spotlight', 'trending', 'new_release', 'staff_pick');

CREATE TABLE discovery_feed (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- User
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,

    -- Item
    item_type feed_item_type NOT NULL,
    story_id UUID REFERENCES story(id) ON DELETE CASCADE,
    author_id UUID REFERENCES author(id) ON DELETE CASCADE,
    collection_id UUID REFERENCES reading_list(id) ON DELETE CASCADE,

    -- Feed Position
    feed_date DATE NOT NULL,
    position INTEGER NOT NULL,

    -- Recommendation Score
    relevance_score DECIMAL(5, 4) NOT NULL,

    -- Status
    is_seen BOOLEAN DEFAULT FALSE,
    seen_at TIMESTAMP WITH TIME ZONE,
    is_clicked BOOLEAN DEFAULT FALSE,
    clicked_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT discovery_feed_user_date_position_unique UNIQUE (user_id, feed_date, position)
);

-- Indexes
CREATE INDEX idx_discovery_feed_user_date ON discovery_feed(user_id, feed_date DESC);
CREATE INDEX idx_discovery_feed_user_unseen ON discovery_feed(user_id, is_seen, position) WHERE is_seen = FALSE;
CREATE INDEX idx_discovery_feed_story ON discovery_feed(story_id) WHERE story_id IS NOT NULL;
```

#### Table: `search_query`

Search history and analytics.

```sql
CREATE TABLE search_query (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- User (NULL = anonymous)
    user_id UUID REFERENCES "user"(id) ON DELETE SET NULL,

    -- Query
    query TEXT NOT NULL,
    normalized_query TEXT NOT NULL,

    -- Filters Applied
    filters JSONB DEFAULT '{}'::JSONB, -- {"genres": ["fantasy"], "status": "completed"}

    -- Results
    result_count INTEGER,
    clicked_story_id UUID REFERENCES story(id) ON DELETE SET NULL,

    -- Performance
    execution_time_ms INTEGER,

    -- Context
    session_id UUID,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_search_queries_user ON search_query(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_search_queries_query ON search_query(normalized_query);
CREATE INDEX idx_search_queries_created ON search_query(created_at DESC);
CREATE INDEX idx_search_queries_popular ON search_query(normalized_query, created_at DESC);
```

### 5.2 API Endpoints

#### Discovery Feed

| Method | Path                                     | Description                     | Auth Required |
| ------ | ---------------------------------------- | ------------------------------- | ------------- |
| GET    | `/api/v1/discover/feed`                  | Get personalized discovery feed | User          |
| GET    | `/api/v1/discover/feed/daily`            | Get daily feed                  | User          |
| POST   | `/api/v1/discover/feed/{itemId}/dismiss` | Dismiss feed item               | User          |
| GET    | `/api/v1/discover/trending`              | Get trending content            | No            |
| GET    | `/api/v1/discover/new-releases`          | Get new releases                | No            |
| GET    | `/api/v1/discover/staff-picks`           | Get staff picks                 | No            |
| GET    | `/api/v1/discover/genres/{genreId}`      | Discover by genre               | No            |
| GET    | `/api/v1/discover/similar/{storyId}`     | Get similar stories             | No            |

#### Recommendations

| Method | Path                                                          | Description                        | Auth Required |
| ------ | ------------------------------------------------------------- | ---------------------------------- | ------------- |
| GET    | `/api/v1/discover/recommendations`                            | Get personalized recommendations   | User          |
| GET    | `/api/v1/discover/recommendations/for-you`                    | Get "For You" recommendations      | User          |
| GET    | `/api/v1/discover/recommendations/because-you-read/{storyId}` | Get "Because You Read"             | User          |
| POST   | `/api/v1/discover/recommendations/{recId}/feedback`           | Provide feedback on recommendation | User          |
| GET    | `/api/v1/discover/recommendations/explain/{recId}`            | Get recommendation explanation     | User          |

#### User Preferences

| Method | Path                                     | Description                   | Auth Required |
| ------ | ---------------------------------------- | ----------------------------- | ------------- |
| GET    | `/api/v1/discover/preferences`           | Get user preferences          | User          |
| PUT    | `/api/v1/discover/preferences`           | Update user preferences       | User          |
| POST   | `/api/v1/discover/preferences/genres`    | Update genre preferences      | User          |
| POST   | `/api/v1/discover/preferences/tags`      | Update tag preferences        | User          |
| POST   | `/api/v1/discover/preferences/reset`     | Reset preferences to defaults | User          |
| GET    | `/api/v1/discover/preferences/suggested` | Get suggested preferences     | User          |

#### Search Enhancement

| Method | Path                         | Description                  | Auth Required |
| ------ | ---------------------------- | ---------------------------- | ------------- |
| GET    | `/api/v1/search/suggestions` | Get search suggestions       | No            |
| GET    | `/api/v1/search/trending`    | Get trending searches        | No            |
| GET    | `/api/v1/search/history`     | Get user's search history    | User          |
| DELETE | `/api/v1/search/history`     | Clear search history         | User          |
| GET    | `/api/v1/search/filters`     | Get available search filters | No            |

### 5.3 Request/Response Examples

#### Get Discovery Feed

**Request:**

```json
GET /api/v1/discover/feed?page=1&pageSize=20
```

**Response:**

```json
{
    "data": {
        "items": [
            {
                "id": "uuid",
                "type": "story",
                "position": 1,
                "story": {
                    "id": "uuid",
                    "title": "The Dragon's Legacy",
                    "author": {
                        "id": "uuid",
                        "stageName": "Sarah Writer"
                    },
                    "synopsis": "An epic fantasy tale...",
                    "genres": ["fantasy", "adventure"],
                    "statistics": {
                        "totalViews": 15000,
                        "averageRating": 4.7
                    }
                },
                "reason": "because_you_read",
                "reasonContext": {
                    "similarStoryId": "uuid",
                    "similarStoryTitle": "The Magic Kingdom"
                },
                "relevanceScore": 0.92
            },
            {
                "id": "uuid",
                "type": "trending",
                "position": 2,
                "stories": [
                    {
                        "id": "uuid",
                        "title": "Trending Story #1",
                        "trendingRank": 1
                    }
                ]
            }
        ],
        "pagination": {
            "page": 1,
            "pageSize": 20,
            "total": 100,
            "hasMore": true
        }
    }
}
```

#### Get Personalized Recommendations

**Request:**

```json
GET /api/v1/discover/recommendations?limit=10&type=all
```

**Response:**

```json
{
    "data": {
        "recommendations": [
            {
                "id": "uuid",
                "type": "similar",
                "reason": "similar_to_favorites",
                "confidenceScore": 0.89,
                "story": {
                    "id": "uuid",
                    "title": "Mystic Realms",
                    "author": {
                        "id": "uuid",
                        "stageName": "Fantasy Author"
                    },
                    "genres": ["fantasy", "magic"],
                    "statistics": {
                        "totalViews": 8200,
                        "averageRating": 4.5,
                        "totalFavorites": 340
                    }
                },
                "explanation": "Based on your favorite story 'The Magic Kingdom' and your interest in fantasy genres"
            }
        ],
        "refreshAvailableAt": "2026-01-14T12:00:00Z"
    }
}
```

#### Update User Preferences

**Request:**

```json
PUT /api/v1/discover/preferences
{
  "preferredGenres": ["uuid1", "uuid2"],
  "preferredStoryLength": "long",
  "notifyNewChapters": true,
  "showMatureContent": false,
  "recommendationDiversity": 0.4
}
```

**Response:**

```json
{
    "data": {
        "userId": "uuid",
        "preferredGenres": [
            { "id": "uuid1", "name": "Fantasy" },
            { "id": "uuid2", "name": "Science Fiction" }
        ],
        "preferredStoryLength": "long",
        "notifyNewChapters": true,
        "showMatureContent": false,
        "recommendationDiversity": 0.4,
        "updatedAt": "2026-01-14T10:30:00Z"
    },
    "message": "Preferences updated successfully"
}
```

### 5.4 Integration Points

- **Recommendation Engine**: ML service queries `user_preference`, `content_similarity` to generate `recommendation` records
- **Search Enhancement**: `search_query` logs feed into autocomplete and trending search features
- **Feed Generation**: Daily job creates personalized `discovery_feed` entries for each user
- **Analytics**: Recommendation performance tracked in `analytics_engagement`
- **Reading History**: `reading_progress` influences recommendation algorithm
- **Real-time Updates**: WebSocket pushes new recommendations as they become available

---

## Summary

### New Tables Summary

| Feature Area       | Tables                                                                                                                                   | Description                                                                    |
| ------------------ | ---------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Reading Experience | `reading_progress`, `reading_list`, `reading_list_item`, `offline_content`                                                               | 4 tables for tracking reading progress, organizing content, and offline access |
| Author Analytics   | `analytics_view`, `analytics_engagement`, `analytics_daily_summary`, `content_performance`                                               | 4 tables for comprehensive analytics tracking                                  |
| Monetization       | `subscription_plan`, `user_subscription`, `payment_transaction`, `premium_content`, `content_purchase`, `revenue_share`, `author_payout` | 7 tables for subscription management, payments, and revenue sharing            |
| Collaboration      | `workspace`, `workspace_member`, `co_author`, `editorial_review`, `editorial_comment`, `writing_session`, `writing_session_participant`  | 7 tables for collaborative writing features                                    |
| Discovery          | `user_preference`, `recommendation`, `content_similarity`, `discovery_feed`, `search_query`                                              | 5 tables for personalized discovery and recommendations                        |

**Total New Tables**: 27
**Total Tables After Implementation**: 49 (22 existing + 27 new)

### New API Endpoints Summary

| Feature Area       | Endpoints | Description                                          |
| ------------------ | --------- | ---------------------------------------------------- |
| Reading Experience | 18        | Progress tracking, reading lists, offline content    |
| Author Analytics   | 12        | Dashboard, reports, exports, real-time stats         |
| Monetization       | 21        | Subscriptions, payments, billing, revenue management |
| Collaboration      | 28        | Workspaces, co-authors, reviews, writing sessions    |
| Discovery          | 18        | Feed, recommendations, preferences, search           |

**Total New Endpoints**: 97
**Total Endpoints After Implementation**: 195 (98 existing + 97 new)

### Integration Checklist

- [ ] Create database migrations for all 27 new tables
- [ ] Implement middleware for reading progress tracking
- [ ] Set up payment provider webhooks (Stripe/PayPal)
- [ ] Configure ML service for recommendations
- [ ] Implement WebSocket handlers for real-time collaboration
- [ ] Create scheduled jobs for analytics aggregation
- [ ] Build admin dashboard for revenue and analytics overview
- [ ] Set up CDN for offline content caching
- [ ] Configure search indexing for enhanced discovery
- [ ] Implement A/B testing framework for recommendation algorithms

### Backward Compatibility

All new features are additive and do not modify existing tables. Existing APIs remain unchanged. New endpoints follow existing patterns:

- Authentication: Cookie-based JWT with scope system
- Routing: `/api/v1/{feature}/`
- Response Format: Consistent with existing API spec
- Error Handling: Standard error response format

### Performance Considerations

1. **Partitioning**: Large tables (`analytics_view`, `analytics_engagement`) should use time-based partitioning
2. **Indexing**: All tables include optimized indexes for common query patterns
3. **Caching**: Redis caching for recommendations, trending content, and user preferences
4. **Materialized Views**: Daily summaries pre-calculated for fast dashboard queries
5. **Background Jobs**: Heavy operations (recommendation generation, analytics aggregation) run asynchronously
6. **CDN**: Offline content and static assets served via CDN
7. **Rate Limiting**: API endpoints have appropriate rate limits to prevent abuse

This comprehensive feature improvement plan positions Samsa competitively with modern writing platforms while maintaining clean architecture and scalability.
