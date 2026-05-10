package db

import (
	"database/sql"
	"fmt"
	"time"

	"aikits/internal/storage"
)

// CallsEdgeInserter generates heuristic CALLS edges for a single file within a BatchWrite
// transaction. Implementations live in internal/kg/lang/ to keep this package language-agnostic.
type CallsEdgeInserter interface {
	InsertCallsEdges(q storage.Querier, repoID, fileID int64) error
}

// RepoRow represents a repo record.
type RepoRow struct {
	ID        int64
	RootPath  string
	Name      string
	CreatedAt string
	UpdatedAt string
}

// FileRow represents a file record.
type FileRow struct {
	ID        int64
	RepoID    int64
	Path      string
	Lang      string
	SHA256    string
	Mtime     int64
	Size      int64
	IndexedAt string
}

// SymbolRow represents a symbol record.
type SymbolRow struct {
	RepoID     int64
	FileID     int64
	Lang       string
	Kind       string
	Name       string
	FQN        string
	Signature  string
	Visibility string
	Doc        string
	BodyHash   string
	StartLine  int
	StartCol   int
	EndLine    int
	EndCol     int
	StartByte  int
	EndByte    int
}

// ExtendsRef maps a class/interface FQN to the simple name of its superclass/superinterface.
// Populated by the Java extractor and stored in the refs table (provenance='extends-heuristic').
type ExtendsRef struct {
	ClassFQN  string
	SuperName string
}

// ImplementsRef maps a class FQN to the simple name of one interface it explicitly implements.
// One ref is emitted per interface listed in the `implements` clause.
// Populated by the Java extractor and stored in the refs table (provenance='implements-heuristic').
type ImplementsRef struct {
	ClassFQN      string
	InterfaceName string // simple name (generics stripped, last component of qualified name)
}

// EdgeRow represents an edge record.
type EdgeRow struct {
	RepoID      int64
	Kind        string
	SrcSymbolID int64
	DstSymbolID int64
	Confidence  float64
	Provenance  string
}

// CallsiteRow represents a callsite record.
type CallsiteRow struct {
	ID               int64
	RepoID           int64
	FileID           int64
	CallerSymbolID   *int64
	CalleeText       string
	StartLine        int
	StartCol         int
	EndLine          int
	EndCol           int
	StartByte        int
	EndByte          int
	ResolvedSymbolID *int64
	Confidence       float64
	Provenance       string
}

// UpsertRepo inserts or updates a repo row, returning its ID.
func UpsertRepo(db *sql.DB, rootPath, name string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(`
		INSERT INTO repos (root_path, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(root_path) DO UPDATE SET name=excluded.name, updated_at=excluded.updated_at
	`, rootPath, name, now, now)
	if err != nil {
		return 0, fmt.Errorf("upsert repo: %w", err)
	}
	// try last insert id; if conflict, look up
	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		if err2 := db.QueryRow("SELECT id FROM repos WHERE root_path=?", rootPath).Scan(&id); err2 != nil {
			return 0, fmt.Errorf("lookup repo id: %w", err2)
		}
	}
	return id, nil
}

// DeleteFileSymbols removes all symbols, callsites, and refs for a file.
func DeleteFileSymbols(db *sql.DB, fileID int64) error {
	if _, err := db.Exec("DELETE FROM symbols WHERE file_id=?", fileID); err != nil {
		return fmt.Errorf("delete symbols for file %d: %w", fileID, err)
	}
	if _, err := db.Exec("DELETE FROM callsites WHERE file_id=?", fileID); err != nil {
		return fmt.Errorf("delete callsites for file %d: %w", fileID, err)
	}
	if _, err := db.Exec("DELETE FROM refs WHERE file_id=?", fileID); err != nil {
		return fmt.Errorf("delete refs for file %d: %w", fileID, err)
	}
	return nil
}

// DeleteFile removes a file row (cascades to symbols/callsites via FK).
func DeleteFile(db *sql.DB, fileID int64) error {
	_, err := db.Exec("DELETE FROM files WHERE id=?", fileID)
	return err
}

// ClearRepoData deletes all indexed data for a repo (files, symbols, edges, callsites, refs)
// via cascading FKs, leaving the repo row itself intact. Used by --full re-index.
func ClearRepoData(db *sql.DB, repoID int64) error {
	// Deleting files cascades to symbols → edges, callsites, refs via ON DELETE CASCADE.
	// Delete edges explicitly first to avoid FK ordering issues with src/dst both cascading.
	for _, q := range []string{
		"DELETE FROM edges WHERE repo_id=?",
		"DELETE FROM files WHERE repo_id=?",
	} {
		if _, err := db.Exec(q, repoID); err != nil {
			return fmt.Errorf("clear repo data: %w", err)
		}
	}
	return nil
}

// UpsertFile inserts or replaces a file row, returning its ID.
func UpsertFile(db *sql.DB, row *FileRow) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if row.IndexedAt == "" {
		row.IndexedAt = now
	}
	res, err := db.Exec(`
		INSERT INTO files (repo_id, path, lang, sha256, mtime, size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, path) DO UPDATE SET
			sha256=excluded.sha256, mtime=excluded.mtime, size=excluded.size, indexed_at=excluded.indexed_at
	`, row.RepoID, row.Path, row.Lang, row.SHA256, row.Mtime, row.Size, row.IndexedAt)
	if err != nil {
		return 0, fmt.Errorf("upsert file: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		if err2 := db.QueryRow("SELECT id FROM files WHERE repo_id=? AND path=?", row.RepoID, row.Path).Scan(&id); err2 != nil {
			return 0, fmt.Errorf("lookup file id: %w", err2)
		}
	}
	return id, nil
}

// GetFileByPath returns the file row for a repo-relative path, or nil if not found.
func GetFileByPath(db *sql.DB, repoID int64, path string) (*FileRow, error) {
	row := &FileRow{}
	err := db.QueryRow(
		"SELECT id, repo_id, path, lang, sha256, mtime, size, indexed_at FROM files WHERE repo_id=? AND path=?",
		repoID, path,
	).Scan(&row.ID, &row.RepoID, &row.Path, &row.Lang, &row.SHA256, &row.Mtime, &row.Size, &row.IndexedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	return row, nil
}

// InsertSymbols batch-inserts symbol rows within a transaction, returning their IDs in order.
func InsertSymbols(q storage.Querier, rows []SymbolRow) ([]int64, error) {
	stmt, err := q.Prepare(`
		INSERT INTO symbols
			(repo_id, file_id, lang, kind, name, fqn, signature, visibility, doc, body_hash,
			 start_line, start_col, end_line, end_col, start_byte, end_byte)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(repo_id, kind, fqn) DO UPDATE SET
			file_id=excluded.file_id, name=excluded.name, signature=excluded.signature,
			visibility=excluded.visibility, doc=excluded.doc, body_hash=excluded.body_hash,
			start_line=excluded.start_line, start_col=excluded.start_col,
			end_line=excluded.end_line, end_col=excluded.end_col,
			start_byte=excluded.start_byte, end_byte=excluded.end_byte
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare insert symbols: %w", err)
	}
	defer stmt.Close()

	ids := make([]int64, len(rows))
	for i, r := range rows {
		fqn := sql.NullString{String: r.FQN, Valid: r.FQN != ""}
		res, err := stmt.Exec(
			r.RepoID, r.FileID, r.Lang, r.Kind, r.Name, fqn, r.Signature, r.Visibility, r.Doc, r.BodyHash,
			r.StartLine, r.StartCol, r.EndLine, r.EndCol, r.StartByte, r.EndByte,
		)
		if err != nil {
			return nil, fmt.Errorf("insert symbol %q: %w", r.FQN, err)
		}
		id, _ := res.LastInsertId()
		if id == 0 && r.FQN != "" {
			_ = q.QueryRow("SELECT id FROM symbols WHERE repo_id=? AND kind=? AND fqn=?", r.RepoID, r.Kind, r.FQN).Scan(&id)
		}
		ids[i] = id
	}
	return ids, nil
}

// InsertEdges batch-inserts edge rows within a transaction.
func InsertEdges(q storage.Querier, rows []EdgeRow) error {
	stmt, err := q.Prepare(`
		INSERT INTO edges (repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert edges: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, r := range rows {
		if _, err := stmt.Exec(r.RepoID, r.Kind, r.SrcSymbolID, r.DstSymbolID, r.Confidence, r.Provenance, now); err != nil {
			return fmt.Errorf("insert edge %s->%s: %w", r.Kind, r.Provenance, err)
		}
	}
	return nil
}

// UpsertSemanticEdges inserts semantic edges (confidence=1.0, provenance=gopls/jdtls),
// upgrading any existing lower-confidence edge for the same (repo, kind, src, dst) pair.
// Use this instead of InsertEdges for LSP-resolved edges so that treesitter CALLS edges
// (confidence=0.4) are upgraded rather than causing a unique-constraint conflict.
func UpsertSemanticEdges(q storage.Querier, rows []EdgeRow) error {
	stmt, err := q.Prepare(`
		INSERT INTO edges (repo_id, kind, src_symbol_id, dst_symbol_id, confidence, provenance, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, kind, src_symbol_id, dst_symbol_id) DO UPDATE SET
			confidence  = excluded.confidence,
			provenance  = excluded.provenance,
			created_at  = excluded.created_at
		WHERE excluded.confidence > edges.confidence
	`)
	if err != nil {
		return fmt.Errorf("prepare upsert semantic edges: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, r := range rows {
		if _, err := stmt.Exec(r.RepoID, r.Kind, r.SrcSymbolID, r.DstSymbolID, r.Confidence, r.Provenance, now); err != nil {
			return fmt.Errorf("upsert semantic edge %s->%s: %w", r.Kind, r.Provenance, err)
		}
	}
	return nil
}

// InsertCallsites batch-inserts callsite rows within a transaction.
func InsertCallsites(q storage.Querier, rows []CallsiteRow) error {
	stmt, err := q.Prepare(`
		INSERT INTO callsites
			(repo_id, file_id, caller_symbol_id, callee_text,
			 start_line, start_col, end_line, end_col, start_byte, end_byte,
			 confidence, provenance)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert callsites: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		callerID := sql.NullInt64{}
		if r.CallerSymbolID != nil {
			callerID = sql.NullInt64{Int64: *r.CallerSymbolID, Valid: true}
		}
		if _, err := stmt.Exec(
			r.RepoID, r.FileID, callerID, r.CalleeText,
			r.StartLine, r.StartCol, r.EndLine, r.EndCol, r.StartByte, r.EndByte,
			r.Confidence, r.Provenance,
		); err != nil {
			return fmt.Errorf("insert callsite %q: %w", r.CalleeText, err)
		}
	}
	return nil
}

// BatchWrite wraps file upsert + symbol + edge + callsite inserts in a single transaction.
// callsInserters are called in order (within the transaction) to generate per-file heuristic
// CALLS edges after caller IDs are populated. Pass nil to skip CALLS edge generation.
func BatchWrite(sqlDB *sql.DB, fileRow *FileRow, symbols []SymbolRow, edges []EdgeRow, callsites []CallsiteRow, callsInserters []CallsEdgeInserter) (fileID int64, symbolIDs []int64, err error) {
	tx, err := sqlDB.Begin()
	if err != nil {
		return 0, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Upsert the file within the transaction
	now := time.Now().UTC().Format(time.RFC3339)
	if fileRow.IndexedAt == "" {
		fileRow.IndexedAt = now
	}
	res, execErr := tx.Exec(`
		INSERT INTO files (repo_id, path, lang, sha256, mtime, size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_id, path) DO UPDATE SET
			sha256=excluded.sha256, mtime=excluded.mtime, size=excluded.size, indexed_at=excluded.indexed_at
	`, fileRow.RepoID, fileRow.Path, fileRow.Lang, fileRow.SHA256, fileRow.Mtime, fileRow.Size, fileRow.IndexedAt)
	if execErr != nil {
		err = fmt.Errorf("upsert file in batch: %w", execErr)
		return
	}
	fileID, _ = res.LastInsertId()
	if fileID == 0 {
		if scanErr := tx.QueryRow("SELECT id FROM files WHERE repo_id=? AND path=?", fileRow.RepoID, fileRow.Path).Scan(&fileID); scanErr != nil {
			err = fmt.Errorf("lookup file id in batch: %w", scanErr)
			return
		}
	}

	// Update fileID on symbols/callsites
	for i := range symbols {
		symbols[i].FileID = fileID
	}
	for i := range callsites {
		callsites[i].FileID = fileID
	}

	// Delete existing data for this file before re-inserting
	for _, q := range []string{
		"DELETE FROM callsites WHERE file_id=?",
		"DELETE FROM refs WHERE file_id=?",
		"DELETE FROM symbols WHERE file_id=?",
	} {
		if _, delErr := tx.Exec(q, fileID); delErr != nil {
			err = fmt.Errorf("cleanup file data: %w", delErr)
			return
		}
	}

	if len(symbols) > 0 {
		symbolIDs, err = InsertSymbols(tx, symbols)
		if err != nil {
			return
		}
	}

	if len(edges) > 0 {
		if err = InsertEdges(tx, edges); err != nil {
			return
		}
	}

	if len(callsites) > 0 {
		if err = InsertCallsites(tx, callsites); err != nil {
			return
		}
		// Populate caller_symbol_id by span containment and generate heuristic CALLS edges.
		if err = populateCallerIDs(tx, fileID); err != nil {
			return
		}
		for _, ins := range callsInserters {
			if err = ins.InsertCallsEdges(tx, fileRow.RepoID, fileID); err != nil {
				return
			}
		}
	}

	err = tx.Commit()
	return
}

// populateCallerIDs sets caller_symbol_id on callsites whose caller is unknown by finding the
// smallest enclosing function/method/arrow_function symbol by byte span.
func populateCallerIDs(q storage.Querier, fileID int64) error {
	_, err := q.Exec(`
		UPDATE callsites SET caller_symbol_id = (
			SELECT s.id FROM symbols s
			WHERE s.file_id = callsites.file_id
			  AND s.kind IN ('function', 'method', 'arrow_function')
			  AND s.start_byte <= callsites.start_byte
			  AND s.end_byte   >= callsites.end_byte
			ORDER BY (s.end_byte - s.start_byte) ASC
			LIMIT 1
		)
		WHERE callsites.file_id = ? AND callsites.caller_symbol_id IS NULL
	`, fileID)
	return err
}

// GetRepoCounts returns summary counts for a repo.
func GetRepoCounts(db *sql.DB, repoID int64) (files, symbols, callsites, resolvedCallsites int64, lastIndexed string, err error) {
	_ = db.QueryRow("SELECT COUNT(*) FROM files WHERE repo_id=?", repoID).Scan(&files)
	_ = db.QueryRow("SELECT COUNT(*) FROM symbols WHERE repo_id=?", repoID).Scan(&symbols)
	_ = db.QueryRow("SELECT COUNT(*) FROM callsites WHERE repo_id=?", repoID).Scan(&callsites)
	_ = db.QueryRow("SELECT COUNT(*) FROM callsites WHERE repo_id=? AND resolved_symbol_id IS NOT NULL", repoID).Scan(&resolvedCallsites)
	_ = db.QueryRow("SELECT COALESCE(MAX(indexed_at),'never') FROM files WHERE repo_id=?", repoID).Scan(&lastIndexed)
	return
}

// ListFilesForRepo returns all file rows for a repo.
func ListFilesForRepo(db *sql.DB, repoID int64) ([]FileRow, error) {
	rows, err := db.Query("SELECT id, repo_id, path, lang, sha256, mtime, size, indexed_at FROM files WHERE repo_id=?", repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []FileRow
	for rows.Next() {
		var r FileRow
		if err := rows.Scan(&r.ID, &r.RepoID, &r.Path, &r.Lang, &r.SHA256, &r.Mtime, &r.Size, &r.IndexedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetRepoByPath returns the repo row for a given root path, or nil if not found.
func GetRepoByPath(db *sql.DB, rootPath string) (*RepoRow, error) {
	row := &RepoRow{}
	err := db.QueryRow(
		"SELECT id, root_path, name, created_at, updated_at FROM repos WHERE root_path=?", rootPath,
	).Scan(&row.ID, &row.RootPath, &row.Name, &row.CreatedAt, &row.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get repo: %w", err)
	}
	return row, nil
}

// GetUnresolvedCallsites returns callsites with no resolved_symbol_id, up to limit.
func GetUnresolvedCallsites(sqlDB *sql.DB, repoID int64, limit int) ([]CallsiteRow, error) {
	rows, err := sqlDB.Query(`
		SELECT id, repo_id, file_id, caller_symbol_id, callee_text,
		       start_line, start_col, end_line, end_col, start_byte, end_byte,
		       resolved_symbol_id, confidence, provenance
		FROM callsites
		WHERE repo_id=? AND resolved_symbol_id IS NULL
		LIMIT ?
	`, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CallsiteRow
	for rows.Next() {
		var r CallsiteRow
		var callerID sql.NullInt64
		var resolvedID sql.NullInt64
		if err := rows.Scan(
			&r.ID, &r.RepoID, &r.FileID, &callerID, &r.CalleeText,
			&r.StartLine, &r.StartCol, &r.EndLine, &r.EndCol, &r.StartByte, &r.EndByte,
			&resolvedID, &r.Confidence, &r.Provenance,
		); err != nil {
			return nil, err
		}
		if callerID.Valid {
			r.CallerSymbolID = &callerID.Int64
		}
		if resolvedID.Valid {
			r.ResolvedSymbolID = &resolvedID.Int64
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// UpdateCallsiteResolved sets the resolved_symbol_id for a callsite.
func UpdateCallsiteResolved(sqlDB *sql.DB, callsiteID, symbolID int64) error {
	_, err := sqlDB.Exec("UPDATE callsites SET resolved_symbol_id=? WHERE id=?", symbolID, callsiteID)
	return err
}

// StoreImportRefs records per-file import paths in the refs table so that
// GenerateImportEdges can rebuild IMPORTS edges correctly.
// srcPkgFQN is the FQN of the package symbol for the importing file (e.g.
// "aikits/internal/kg/db" for Go or "com.example" for Java).
// Existing refs for this file are expected to have been cleared by BatchWrite.
func StoreImportRefs(sqlDB *sql.DB, repoID, fileID int64, srcPkgFQN string, importPaths []string) error {
	if len(importPaths) == 0 {
		return nil
	}
	var srcID int64
	if err := sqlDB.QueryRow(
		"SELECT id FROM symbols WHERE repo_id=? AND kind='package' AND fqn=?",
		repoID, srcPkgFQN,
	).Scan(&srcID); err != nil {
		return nil // package symbol not in DB yet — skip
	}

	stmt, err := sqlDB.Prepare(`
		INSERT INTO refs
			(repo_id, file_id, src_symbol_id, ref_text,
			 start_line, start_col, end_line, end_col, start_byte, end_byte,
			 confidence, provenance)
		VALUES (?,?,?,?,0,0,0,0,0,0,0.8,'extractor')
	`)
	if err != nil {
		return fmt.Errorf("prepare store import refs: %w", err)
	}
	defer stmt.Close()

	for _, path := range importPaths {
		if _, err := stmt.Exec(repoID, fileID, srcID, path); err != nil {
			return fmt.Errorf("insert import ref %q: %w", path, err)
		}
	}
	return nil
}

// StoreExtendsRefs records per-file Java extends declarations in the refs table so that
// GenerateExtendsEdges can rebuild EXTENDS edges correctly.
// classFQN is the FQN of the extending class (e.g. "com.example.MyClass").
// Existing refs for this file are expected to have been cleared by BatchWrite (via DeleteFileSymbols).
func StoreExtendsRefs(sqlDB *sql.DB, repoID, fileID int64, refs []ExtendsRef) error {
	if len(refs) == 0 {
		return nil
	}

	stmt, err := sqlDB.Prepare(`
		INSERT INTO refs
			(repo_id, file_id, src_symbol_id, ref_text,
			 start_line, start_col, end_line, end_col, start_byte, end_byte,
			 confidence, provenance)
		VALUES (?,?,?,?,0,0,0,0,0,0,0.5,'extends-heuristic')
	`)
	if err != nil {
		return fmt.Errorf("prepare store extends refs: %w", err)
	}
	defer stmt.Close()

	for _, ref := range refs {
		var srcID int64
		if err := sqlDB.QueryRow(
			"SELECT id FROM symbols WHERE repo_id=? AND lang='java' AND fqn=?",
			repoID, ref.ClassFQN,
		).Scan(&srcID); err != nil {
			continue // class symbol not indexed yet — skip
		}
		if _, err := stmt.Exec(repoID, fileID, srcID, ref.SuperName); err != nil {
			return fmt.Errorf("insert extends ref %q -> %q: %w", ref.ClassFQN, ref.SuperName, err)
		}
	}
	return nil
}

// StoreImplementsRefs records per-file Java implements declarations in the refs table so that
// InsertImplementsEdges can rebuild IMPLEMENTS edges correctly.
// One ref per interface listed in the `implements` clause is expected.
// Existing refs for this file are expected to have been cleared by BatchWrite (via DeleteFileSymbols).
func StoreImplementsRefs(sqlDB *sql.DB, repoID, fileID int64, refs []ImplementsRef) error {
	if len(refs) == 0 {
		return nil
	}

	stmt, err := sqlDB.Prepare(`
		INSERT INTO refs
			(repo_id, file_id, src_symbol_id, ref_text,
			 start_line, start_col, end_line, end_col, start_byte, end_byte,
			 confidence, provenance)
		VALUES (?,?,?,?,0,0,0,0,0,0,0.5,'implements-heuristic')
	`)
	if err != nil {
		return fmt.Errorf("prepare store implements refs: %w", err)
	}
	defer stmt.Close()

	for _, ref := range refs {
		var srcID int64
		if err := sqlDB.QueryRow(
			"SELECT id FROM symbols WHERE repo_id=? AND lang='java' AND fqn=?",
			repoID, ref.ClassFQN,
		).Scan(&srcID); err != nil {
			continue // class symbol not indexed yet — skip
		}
		if _, err := stmt.Exec(repoID, fileID, srcID, ref.InterfaceName); err != nil {
			return fmt.Errorf("insert implements ref %q -> %q: %w", ref.ClassFQN, ref.InterfaceName, err)
		}
	}
	return nil
}

// TypeRef maps a source symbol FQN to a fully-qualified referenced type name.
// TypeName must be a resolved FQN (not a bare name) to enable exact SQL matching.
type TypeRef struct {
	SrcFQN   string // FQN of the symbol that references the type
	TypeName string // fully-qualified name of the referenced type
}

// StoreTypeRefs records per-file type references in the refs table so that
// GenerateReferencesEdges can rebuild REFERENCES edges correctly.
// TypeName in each TypeRef must be a resolved FQN (not a bare name) to ensure
// exact matching against dst.fqn in SQL. Existing refs for this file are
// expected to have been cleared by BatchWrite (via DeleteFileSymbols).
func StoreTypeRefs(sqlDB *sql.DB, repoID, fileID int64, refs []TypeRef) error {
	if len(refs) == 0 {
		return nil
	}
	stmt, err := sqlDB.Prepare(`
		INSERT OR IGNORE INTO refs
			(repo_id, file_id, src_symbol_id, ref_text,
			 start_line, start_col, end_line, end_col, start_byte, end_byte,
			 confidence, provenance)
		VALUES (?,?,?,?,0,0,0,0,0,0,0.4,'type-ref')
	`)
	if err != nil {
		return fmt.Errorf("prepare store type refs: %w", err)
	}
	defer stmt.Close()

	for _, ref := range refs {
		var srcID int64
		if err := sqlDB.QueryRow(
			"SELECT id FROM symbols WHERE repo_id=? AND fqn=?",
			repoID, ref.SrcFQN,
		).Scan(&srcID); err != nil {
			continue // source symbol not indexed yet — skip silently
		}
		if _, err := stmt.Exec(repoID, fileID, srcID, ref.TypeName); err != nil {
			return fmt.Errorf("insert type ref %q -> %q: %w", ref.SrcFQN, ref.TypeName, err)
		}
	}
	return nil
}
