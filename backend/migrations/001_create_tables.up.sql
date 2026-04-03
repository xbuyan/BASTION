-- 001_create_tables.up.sql
-- Run with: golang-migrate or goose

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ─── Users ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(100) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    xp            INTEGER DEFAULT 0,
    streak        INTEGER DEFAULT 0,
    last_active   TIMESTAMP,
    created_at    TIMESTAMP DEFAULT NOW()
);

-- ─── Exercises ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS exercises (
    id                   SERIAL PRIMARY KEY,
    phase                INTEGER NOT NULL,
    num                  VARCHAR(10) NOT NULL,
    title                VARCHAR(255) NOT NULL,
    domain               VARCHAR(50) NOT NULL,
    difficulty           INTEGER NOT NULL CHECK (difficulty BETWEEN 1 AND 3),
    tags                 JSONB DEFAULT '[]',
    description          TEXT NOT NULL DEFAULT '',
    starter_code         TEXT DEFAULT '',
    test_cases           JSONB DEFAULT '{}',
    hints                JSONB DEFAULT '[]',
    solution_explanation TEXT DEFAULT '',
    time_complexity      VARCHAR(50) DEFAULT '',
    space_complexity     VARCHAR(50) DEFAULT '',
    concepts_covered     JSONB DEFAULT '[]',
    related_exercises    JSONB DEFAULT '[]',
    created_at           TIMESTAMP DEFAULT NOW()
);

-- ─── User Exercise Progress ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_exercises (
    user_id      UUID REFERENCES users(id) ON DELETE CASCADE,
    exercise_id  INTEGER REFERENCES exercises(id) ON DELETE CASCADE,
    status       VARCHAR(20) DEFAULT 'not_started'
                 CHECK (status IN ('not_started','in_progress','completed')),
    code         TEXT DEFAULT '',
    notes        TEXT DEFAULT '',
    completed_at TIMESTAMP,
    PRIMARY KEY (user_id, exercise_id)
);

-- ─── Spaced Repetition ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS review_cards (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID REFERENCES users(id) ON DELETE CASCADE,
    item_type    VARCHAR(20) NOT NULL CHECK (item_type IN ('exercise','concept','paper')),
    item_id      VARCHAR(255) NOT NULL,
    interval     INTEGER DEFAULT 1,
    ease_factor  DECIMAL(4,2) DEFAULT 2.5,
    repetitions  INTEGER DEFAULT 0,
    next_review  TIMESTAMP NOT NULL DEFAULT NOW(),
    last_review  TIMESTAMP,
    created_at   TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_review_cards_user_due ON review_cards (user_id, next_review);

-- ─── Knowledge Graph ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS concepts (
    id          VARCHAR(50) PRIMARY KEY,
    label       VARCHAR(255) NOT NULL,
    domain      VARCHAR(50) NOT NULL,
    phase       INTEGER NOT NULL,
    description TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS concept_edges (
    concept_id      VARCHAR(50) REFERENCES concepts(id) ON DELETE CASCADE,
    prerequisite_id VARCHAR(50) REFERENCES concepts(id) ON DELETE CASCADE,
    PRIMARY KEY (concept_id, prerequisite_id)
);

CREATE TABLE IF NOT EXISTS user_concepts (
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    concept_id    VARCHAR(50) REFERENCES concepts(id) ON DELETE CASCADE,
    mastery_level INTEGER DEFAULT 0 CHECK (mastery_level BETWEEN 0 AND 2),
    PRIMARY KEY (user_id, concept_id)
);

-- ─── Projects ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS projects (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    phase       INTEGER NOT NULL,
    status      VARCHAR(20) DEFAULT 'concept'
                CHECK (status IN ('concept','prototype','production','published')),
    github_repo VARCHAR(255) DEFAULT '',
    milestones  JSONB DEFAULT '[]',
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP
);

-- ─── Papers ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS papers (
    id         VARCHAR(50) PRIMARY KEY,
    title      VARCHAR(500) NOT NULL,
    authors    TEXT DEFAULT '',
    year       INTEGER,
    venue      VARCHAR(255) DEFAULT '',
    tags       JSONB DEFAULT '[]',
    abstract   TEXT DEFAULT '',
    url        VARCHAR(500) DEFAULT '',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_papers (
    user_id             UUID REFERENCES users(id) ON DELETE CASCADE,
    paper_id            VARCHAR(50) REFERENCES papers(id) ON DELETE CASCADE,
    status              VARCHAR(20) DEFAULT 'queued'
                        CHECK (status IN ('queued','reading','read')),
    notes               TEXT DEFAULT '',
    questions           TEXT DEFAULT '',
    implementation_link TEXT DEFAULT '',
    PRIMARY KEY (user_id, paper_id)
);

-- ─── Contributions ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS contributions (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID REFERENCES users(id) ON DELETE CASCADE,
    repo           VARCHAR(255) NOT NULL,
    title          VARCHAR(500) NOT NULL,
    description    TEXT DEFAULT '',
    url            VARCHAR(500) DEFAULT '',
    status         VARCHAR(20) DEFAULT 'open'
                   CHECK (status IN ('open','merged','closed')),
    skills         JSONB DEFAULT '[]',
    contributed_at TIMESTAMP,
    synced_at      TIMESTAMP
);

-- ─── Research Workspace ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS hypotheses (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    status      VARCHAR(20) DEFAULT 'exploring'
                CHECK (status IN ('exploring','validated','rejected')),
    experiments JSONB DEFAULT '[]',
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS experiments (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID REFERENCES users(id) ON DELETE CASCADE,
    hypothesis_id  UUID REFERENCES hypotheses(id) ON DELETE SET NULL,
    title          VARCHAR(255) NOT NULL,
    description    TEXT DEFAULT '',
    result         TEXT DEFAULT '',
    status         VARCHAR(20) DEFAULT 'running'
                   CHECK (status IN ('running','complete','failed')),
    conducted_at   TIMESTAMP,
    created_at     TIMESTAMP DEFAULT NOW()
);

-- ─── Sessions & Analytics ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_activity (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    activity_data JSONB DEFAULT '{}',
    xp_gained     INTEGER DEFAULT 0,
    created_at    TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_activity_user ON user_activity (user_id, created_at DESC);
