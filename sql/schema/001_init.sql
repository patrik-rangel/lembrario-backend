CREATE TABLE contents (
    id VARCHAR(26) PRIMARY KEY,
    url TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    type VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE metadata (
    content_id VARCHAR(26) PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
    title TEXT,
    description TEXT,
    thumbnail_path TEXT,
    transcript TEXT,
    provider VARCHAR(50),
    reading_time INT,
    raw_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notes (
    id VARCHAR(26) PRIMARY KEY,
    content_id VARCHAR(26) UNIQUE NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tags (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE contents_tags (
    content_id VARCHAR(26) NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    tag_id VARCHAR(26) NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (content_id, tag_id)
);
