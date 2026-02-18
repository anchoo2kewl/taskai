-- Add wiki tables for collaborative documentation
-- SQLite version

-- Wiki pages table
CREATE TABLE wiki_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    created_by INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, slug),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Yjs updates (append-only log for CRDT)
CREATE TABLE yjs_updates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    update_data BLOB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER,
    FOREIGN KEY (page_id) REFERENCES wiki_pages(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Page snapshots for faster loading
CREATE TABLE page_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    version_number INTEGER NOT NULL,
    yjs_state BLOB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(page_id, version_number),
    FOREIGN KEY (page_id) REFERENCES wiki_pages(id) ON DELETE CASCADE
);

-- Content blocks for full-text search
CREATE TABLE wiki_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,
    block_type TEXT NOT NULL,
    level INTEGER,
    headings_path TEXT,
    canonical_json TEXT,
    plain_text TEXT,
    position INTEGER NOT NULL,
    search_text TEXT,
    FOREIGN KEY (page_id) REFERENCES wiki_pages(id) ON DELETE CASCADE
);

-- Indexes for wiki tables
CREATE INDEX idx_wiki_pages_project_id ON wiki_pages(project_id);
CREATE INDEX idx_wiki_pages_slug ON wiki_pages(slug);
CREATE INDEX idx_yjs_updates_page_id ON yjs_updates(page_id);
CREATE INDEX idx_page_versions_page_id ON page_versions(page_id);
CREATE INDEX idx_wiki_blocks_page_id ON wiki_blocks(page_id);

-- SQLite FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE wiki_blocks_fts USING fts5(
    plain_text,
    headings_path,
    content='wiki_blocks',
    content_rowid='id'
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER wiki_blocks_ai AFTER INSERT ON wiki_blocks BEGIN
    INSERT INTO wiki_blocks_fts(rowid, plain_text, headings_path)
    VALUES (new.id, new.plain_text, new.headings_path);
END;

CREATE TRIGGER wiki_blocks_ad AFTER DELETE ON wiki_blocks BEGIN
    INSERT INTO wiki_blocks_fts(wiki_blocks_fts, rowid, plain_text, headings_path)
    VALUES('delete', old.id, old.plain_text, old.headings_path);
END;

CREATE TRIGGER wiki_blocks_au AFTER UPDATE ON wiki_blocks BEGIN
    INSERT INTO wiki_blocks_fts(wiki_blocks_fts, rowid, plain_text, headings_path)
    VALUES('delete', old.id, old.plain_text, old.headings_path);
    INSERT INTO wiki_blocks_fts(rowid, plain_text, headings_path)
    VALUES (new.id, new.plain_text, new.headings_path);
END;
