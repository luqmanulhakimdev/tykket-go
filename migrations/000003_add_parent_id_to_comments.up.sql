ALTER TABLE comments ADD COLUMN IF NOT EXISTS parent_id UUID NULL REFERENCES comments(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

