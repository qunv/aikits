-- Migration 001: Initial schema
-- Creates the core knowledge table and FTS5 index.

CREATE TABLE IF NOT EXISTS knowledge (
  id               TEXT PRIMARY KEY,
  title            TEXT NOT NULL CHECK (length(title) <= 100),
  content          TEXT NOT NULL CHECK (length(content) <= 5000),
  tags             TEXT NOT NULL DEFAULT '[]',
  scope            TEXT NOT NULL DEFAULT 'global',
  normalized_title TEXT NOT NULL,
  content_hash     TEXT NOT NULL,
  created_at       TEXT NOT NULL,
  updated_at       TEXT NOT NULL,

  UNIQUE (normalized_title, scope),
  UNIQUE (content_hash, scope)
);

-- FTS5 virtual table: title weighted ×10, content ×5, tags ×1
CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(
  title,
  content,
  tags,
  content='knowledge',
  content_rowid='rowid',
  tokenize='porter unicode61'
);

-- Keep FTS in sync with the knowledge table.
CREATE TRIGGER IF NOT EXISTS knowledge_ai AFTER INSERT ON knowledge BEGIN
  INSERT INTO knowledge_fts(rowid, title, content, tags)
  VALUES (NEW.rowid, NEW.title, NEW.content, NEW.tags);
END;

CREATE TRIGGER IF NOT EXISTS knowledge_ad AFTER DELETE ON knowledge BEGIN
  INSERT INTO knowledge_fts(knowledge_fts, rowid, title, content, tags)
  VALUES ('delete', OLD.rowid, OLD.title, OLD.content, OLD.tags);
END;

CREATE TRIGGER IF NOT EXISTS knowledge_au AFTER UPDATE ON knowledge BEGIN
  INSERT INTO knowledge_fts(knowledge_fts, rowid, title, content, tags)
  VALUES ('delete', OLD.rowid, OLD.title, OLD.content, OLD.tags);
  INSERT INTO knowledge_fts(rowid, title, content, tags)
  VALUES (NEW.rowid, NEW.title, NEW.content, NEW.tags);
END;

-- Fast scope filtering index.
CREATE INDEX IF NOT EXISTS idx_knowledge_scope ON knowledge(scope);
