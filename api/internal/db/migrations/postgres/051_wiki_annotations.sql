CREATE TABLE IF NOT EXISTS wiki_annotations (
    id BIGSERIAL PRIMARY KEY,
    wiki_page_id BIGINT NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_offset INTEGER NOT NULL,
    end_offset INTEGER NOT NULL,
    selected_text TEXT NOT NULL,
    color VARCHAR(20) NOT NULL DEFAULT 'yellow',
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wiki_annotations_page_id ON wiki_annotations(wiki_page_id);
CREATE INDEX IF NOT EXISTS idx_wiki_annotations_author_id ON wiki_annotations(author_id);

CREATE TABLE IF NOT EXISTS wiki_annotation_comments (
    id BIGSERIAL PRIMARY KEY,
    annotation_id BIGINT NOT NULL REFERENCES wiki_annotations(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id BIGINT REFERENCES wiki_annotation_comments(id) ON DELETE SET NULL,
    content TEXT NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wiki_annotation_comments_annotation_id ON wiki_annotation_comments(annotation_id);
