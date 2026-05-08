PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS repos (
  id        INTEGER PRIMARY KEY,
  root_path TEXT NOT NULL UNIQUE,
  name      TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
  id         INTEGER PRIMARY KEY,
  repo_id    INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  path       TEXT NOT NULL, -- repo-relative, forward-slash normalized
  lang       TEXT NOT NULL, -- go|java
  sha256     TEXT NOT NULL,
  mtime      INTEGER NOT NULL, -- unix seconds
  size       INTEGER NOT NULL,
  indexed_at TEXT NOT NULL,
  UNIQUE(repo_id, path)
);

CREATE TABLE IF NOT EXISTS symbols (
  id          INTEGER PRIMARY KEY,
  repo_id     INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  file_id     INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  lang        TEXT NOT NULL,
  kind        TEXT NOT NULL, -- package|type|interface|method|function|field|var|const|constructor
  name        TEXT NOT NULL,
  fqn         TEXT,
  signature   TEXT,
  visibility  TEXT,
  doc         TEXT,
  body_hash   TEXT,
  start_line  INTEGER NOT NULL,
  start_col   INTEGER NOT NULL,
  end_line    INTEGER NOT NULL,
  end_col     INTEGER NOT NULL,
  start_byte  INTEGER NOT NULL,
  end_byte    INTEGER NOT NULL,
  UNIQUE(repo_id, kind, fqn)
);

CREATE TABLE IF NOT EXISTS edges (
  id             INTEGER PRIMARY KEY,
  repo_id        INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  kind           TEXT NOT NULL, -- CONTAINS|DECLARES|IMPORTS|CALLS|REFERENCES|IMPLEMENTS|EXTENDS|OVERRIDES|READS|WRITES|TESTS
  src_symbol_id  INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
  dst_symbol_id  INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
  confidence     REAL NOT NULL,
  provenance     TEXT NOT NULL,
  created_at     TEXT NOT NULL,
  UNIQUE(repo_id, kind, src_symbol_id, dst_symbol_id)
);

CREATE TABLE IF NOT EXISTS callsites (
  id                INTEGER PRIMARY KEY,
  repo_id           INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  file_id           INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  caller_symbol_id  INTEGER REFERENCES symbols(id) ON DELETE SET NULL,
  callee_text       TEXT NOT NULL,
  start_line        INTEGER NOT NULL,
  start_col         INTEGER NOT NULL,
  end_line          INTEGER NOT NULL,
  end_col           INTEGER NOT NULL,
  start_byte        INTEGER NOT NULL,
  end_byte          INTEGER NOT NULL,
  resolved_symbol_id INTEGER REFERENCES symbols(id) ON DELETE SET NULL,
  confidence        REAL NOT NULL,
  provenance        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS refs (
  id             INTEGER PRIMARY KEY,
  repo_id        INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
  file_id        INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  src_symbol_id  INTEGER REFERENCES symbols(id) ON DELETE SET NULL,
  dst_symbol_id  INTEGER REFERENCES symbols(id) ON DELETE SET NULL,
  ref_text       TEXT,
  start_line     INTEGER NOT NULL,
  start_col      INTEGER NOT NULL,
  end_line       INTEGER NOT NULL,
  end_col        INTEGER NOT NULL,
  start_byte     INTEGER NOT NULL,
  end_byte       INTEGER NOT NULL,
  confidence     REAL NOT NULL,
  provenance     TEXT NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_files_repo_path        ON files(repo_id, path);
CREATE INDEX IF NOT EXISTS idx_symbols_repo_fqn       ON symbols(repo_id, fqn);
CREATE INDEX IF NOT EXISTS idx_symbols_repo_name_kind ON symbols(repo_id, name, kind);
CREATE INDEX IF NOT EXISTS idx_symbols_file_span      ON symbols(file_id, start_byte, end_byte);
CREATE INDEX IF NOT EXISTS idx_edges_repo_src_kind    ON edges(repo_id, src_symbol_id, kind);
CREATE INDEX IF NOT EXISTS idx_edges_repo_dst_kind    ON edges(repo_id, dst_symbol_id, kind);
CREATE INDEX IF NOT EXISTS idx_callsites_repo_caller  ON callsites(repo_id, caller_symbol_id);
CREATE INDEX IF NOT EXISTS idx_callsites_repo_resolved ON callsites(repo_id, resolved_symbol_id);
CREATE INDEX IF NOT EXISTS idx_callsites_file_span    ON callsites(file_id, start_byte, end_byte);

-- FTS5 virtual tables for full-text symbol and callsite search.
-- External-content tables: index only; content is read from the source tables.
CREATE VIRTUAL TABLE IF NOT EXISTS symbols_fts USING fts5(
    name, fqn, doc,
    content='symbols', content_rowid='id'
);

CREATE VIRTUAL TABLE IF NOT EXISTS callsites_fts USING fts5(
    callee_text,
    content='callsites', content_rowid='id'
);

-- Triggers to keep symbols_fts in sync.
CREATE TRIGGER IF NOT EXISTS symbols_fts_ai AFTER INSERT ON symbols BEGIN
    INSERT INTO symbols_fts(rowid, name, fqn, doc)
    VALUES (new.id, new.name, COALESCE(new.fqn,''), COALESCE(new.doc,''));
END;
CREATE TRIGGER IF NOT EXISTS symbols_fts_ad AFTER DELETE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid, name, fqn, doc)
    VALUES ('delete', old.id, old.name, COALESCE(old.fqn,''), COALESCE(old.doc,''));
END;
CREATE TRIGGER IF NOT EXISTS symbols_fts_au AFTER UPDATE ON symbols BEGIN
    INSERT INTO symbols_fts(symbols_fts, rowid, name, fqn, doc)
    VALUES ('delete', old.id, old.name, COALESCE(old.fqn,''), COALESCE(old.doc,''));
    INSERT INTO symbols_fts(rowid, name, fqn, doc)
    VALUES (new.id, new.name, COALESCE(new.fqn,''), COALESCE(new.doc,''));
END;

-- Triggers to keep callsites_fts in sync.
CREATE TRIGGER IF NOT EXISTS callsites_fts_ai AFTER INSERT ON callsites BEGIN
    INSERT INTO callsites_fts(rowid, callee_text) VALUES (new.id, new.callee_text);
END;
CREATE TRIGGER IF NOT EXISTS callsites_fts_ad AFTER DELETE ON callsites BEGIN
    INSERT INTO callsites_fts(callsites_fts, rowid, callee_text)
    VALUES ('delete', old.id, old.callee_text);
END;
CREATE TRIGGER IF NOT EXISTS callsites_fts_au AFTER UPDATE ON callsites BEGIN
    INSERT INTO callsites_fts(callsites_fts, rowid, callee_text)
    VALUES ('delete', old.id, old.callee_text);
    INSERT INTO callsites_fts(rowid, callee_text) VALUES (new.id, new.callee_text);
END;
