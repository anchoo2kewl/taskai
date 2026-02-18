-- Add wiki tables for collaborative documentation
-- Postgres version

-- Wiki pages table
CREATE TABLE wiki_pages (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(500) NOT NULL,
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, slug)
);

-- Yjs updates (append-only log for CRDT)
CREATE TABLE yjs_updates (
    id BIGSERIAL PRIMARY KEY,
    page_id BIGINT NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    update_data BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by BIGINT REFERENCES users(id)
);

-- Page snapshots for faster loading
CREATE TABLE page_versions (
    id BIGSERIAL PRIMARY KEY,
    page_id BIGINT NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    yjs_state BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(page_id, version_number)
);

-- Content blocks for full-text search
CREATE TABLE wiki_blocks (
    id BIGSERIAL PRIMARY KEY,
    page_id BIGINT NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    block_type VARCHAR(50) NOT NULL,
    level INTEGER,
    headings_path TEXT,
    canonical_json JSONB,
    plain_text TEXT,
    position INTEGER NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(headings_path, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(plain_text, '')), 'B')
    ) STORED
);

-- Indexes for wiki tables
CREATE INDEX idx_wiki_pages_project_id ON wiki_pages(project_id);
CREATE INDEX idx_wiki_pages_slug ON wiki_pages(slug);
CREATE INDEX idx_yjs_updates_page_id ON yjs_updates(page_id);
CREATE INDEX idx_page_versions_page_id ON page_versions(page_id);
CREATE INDEX idx_wiki_blocks_page_id ON wiki_blocks(page_id);
CREATE INDEX idx_wiki_blocks_search ON wiki_blocks USING GIN(search_vector);

-- Enable pg_trgm extension for fuzzy text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;
