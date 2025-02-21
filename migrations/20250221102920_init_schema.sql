CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    allow_comments BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    parent_id UUID NULL REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (LENGTH(content) <= 2000),
    created_at TIMESTAMP DEFAULT NOW()
);
