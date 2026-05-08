package db_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kgdb "aikits/internal/kg/db"
	kglang "aikits/internal/kg/lang"
)

func tmpDB(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sqlite")
	return path, func() { os.Remove(path) }
}

func TestOpenAndSchema(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, err := kgdb.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// second open should succeed (existing schema)
	db2, err := kgdb.Open(path)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	db2.Close()
}

func TestUpsertRepo(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	id, err := kgdb.UpsertRepo(db, "/repo/root", "myrepo")
	if err != nil {
		t.Fatalf("UpsertRepo: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero repo ID")
	}

	// Second upsert should return same ID
	id2, err := kgdb.UpsertRepo(db, "/repo/root", "myrepo-renamed")
	if err != nil {
		t.Fatalf("UpsertRepo second: %v", err)
	}
	if id2 != id {
		t.Fatalf("expected same ID %d, got %d", id, id2)
	}
}

func TestUpsertAndGetFile(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	repoID, _ := kgdb.UpsertRepo(db, "/repo/root", "repo")

	fileRow := &kgdb.FileRow{
		RepoID: repoID,
		Path:   "internal/foo/bar.go",
		Lang:   "go",
		SHA256: "abc123",
		Mtime:  1000,
		Size:   512,
	}

	fid, err := kgdb.UpsertFile(db, fileRow)
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}
	if fid == 0 {
		t.Fatal("expected non-zero file ID")
	}

	got, err := kgdb.GetFileByPath(db, repoID, "internal/foo/bar.go")
	if err != nil {
		t.Fatalf("GetFileByPath: %v", err)
	}
	if got == nil {
		t.Fatal("expected file row, got nil")
	}
	if got.SHA256 != "abc123" {
		t.Errorf("SHA256: want abc123, got %s", got.SHA256)
	}
}

func TestBatchWrite(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	repoID, _ := kgdb.UpsertRepo(db, "/repo/root", "repo")

	fileRow := &kgdb.FileRow{
		RepoID: repoID,
		Path:   "cmd/main.go",
		Lang:   "go",
		SHA256: "deadbeef",
		Mtime:  2000,
		Size:   1024,
	}

	symbols := []kgdb.SymbolRow{
		{
			RepoID: repoID, Lang: "go", Kind: "function",
			Name: "main", FQN: "github.com/example/repo/cmd.main",
			StartLine: 1, StartCol: 1, EndLine: 10, EndCol: 1,
		},
	}
	callsites := []kgdb.CallsiteRow{
		{
			RepoID: repoID, CalleeText: "fmt.Println",
			StartLine: 5, StartCol: 2, EndLine: 5, EndCol: 15,
			Confidence: 0.9, Provenance: "go/ast",
		},
	}

	fileID, symIDs, err := kgdb.BatchWrite(db, fileRow, symbols, nil, callsites, nil)
	if err != nil {
		t.Fatalf("BatchWrite: %v", err)
	}
	if fileID == 0 {
		t.Fatal("expected non-zero fileID")
	}
	if len(symIDs) != 1 {
		t.Fatalf("expected 1 symbol ID, got %d", len(symIDs))
	}

	files, syms, cs, _, _, _ := kgdb.GetRepoCounts(db, repoID)
	if files != 1 {
		t.Errorf("files: want 1, got %d", files)
	}
	if syms != 1 {
		t.Errorf("symbols: want 1, got %d", syms)
	}
	if cs != 1 {
		t.Errorf("callsites: want 1, got %d", cs)
	}
}

func TestGetRepoByPath(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	_, _ = kgdb.UpsertRepo(db, "/repo/root", "myrepo")

	repo, err := kgdb.GetRepoByPath(db, "/repo/root")
	if err != nil {
		t.Fatalf("GetRepoByPath: %v", err)
	}
	if repo == nil {
		t.Fatal("expected repo row, got nil")
	}
	if repo.Name != "myrepo" {
		t.Errorf("name: want myrepo, got %s", repo.Name)
	}

	missing, err := kgdb.GetRepoByPath(db, "/nonexistent")
	if err != nil {
		t.Fatalf("GetRepoByPath missing: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for missing repo")
	}
}

func TestDeleteFile(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	repoID, _ := kgdb.UpsertRepo(db, "/repo/root", "repo")
	fid, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
		RepoID: repoID, Path: "x.go", Lang: "go",
		SHA256: "aa", Mtime: 1, Size: 1,
	})

	if err := kgdb.DeleteFile(db, fid); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}

	got, _ := kgdb.GetFileByPath(db, repoID, "x.go")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestSchemaVersionError(t *testing.T) {
err := &kgdb.SchemaVersionError{Got: 1, Want: 3}
msg := err.Error()
if !strings.Contains(msg, "1") {
t.Errorf("Error() should contain got-version '1': %s", msg)
}
if !strings.Contains(msg, "3") {
t.Errorf("Error() should contain want-version '3': %s", msg)
}
}

func TestSchemaVersionMismatch(t *testing.T) {
dir := t.TempDir()
path := filepath.Join(dir, "v1.sqlite")

rawDB, err := sql.Open("sqlite", path)
if err != nil {
t.Fatalf("sql.Open: %v", err)
}
if _, err := rawDB.Exec("CREATE TABLE schema_version (version INTEGER NOT NULL)"); err != nil {
t.Fatalf("create schema_version: %v", err)
}
if _, err := rawDB.Exec("INSERT INTO schema_version VALUES(1)"); err != nil {
t.Fatalf("insert version: %v", err)
}
rawDB.Close()

_, err = kgdb.Open(path)
if err == nil {
t.Fatal("expected SchemaVersionError, got nil")
}
var sve *kgdb.SchemaVersionError
if !errors.As(err, &sve) {
t.Fatalf("expected *SchemaVersionError, got: %T: %v", err, err)
}
if sve.Got != 1 || sve.Want != 3 {
t.Errorf("SchemaVersionError: Got=%d Want=%d", sve.Got, sve.Want)
}
}

func TestMigrateV2ToV3(t *testing.T) {
dir := t.TempDir()
path := filepath.Join(dir, "v2.sqlite")

v2DDL := []string{
`CREATE TABLE schema_version (version INTEGER NOT NULL)`,
`INSERT INTO schema_version VALUES(2)`,
`CREATE TABLE repos (id INTEGER PRIMARY KEY, root_path TEXT NOT NULL UNIQUE, name TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
`CREATE TABLE files (id INTEGER PRIMARY KEY, repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE, path TEXT NOT NULL, lang TEXT NOT NULL, sha256 TEXT NOT NULL, mtime INTEGER NOT NULL, size INTEGER NOT NULL, indexed_at TEXT NOT NULL, UNIQUE(repo_id, path))`,
`CREATE TABLE symbols (id INTEGER PRIMARY KEY, repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, lang TEXT NOT NULL, kind TEXT NOT NULL, name TEXT NOT NULL, fqn TEXT, signature TEXT, visibility TEXT, doc TEXT, body_hash TEXT, start_line INTEGER NOT NULL, start_col INTEGER NOT NULL, end_line INTEGER NOT NULL, end_col INTEGER NOT NULL, start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL, UNIQUE(repo_id, kind, fqn))`,
`CREATE TABLE edges (id INTEGER PRIMARY KEY, repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE, kind TEXT NOT NULL, src_symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE, dst_symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE, confidence REAL NOT NULL, provenance TEXT NOT NULL, created_at TEXT NOT NULL, UNIQUE(repo_id, kind, src_symbol_id, dst_symbol_id))`,
`CREATE TABLE callsites (id INTEGER PRIMARY KEY, repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, caller_symbol_id INTEGER REFERENCES symbols(id) ON DELETE SET NULL, callee_text TEXT NOT NULL, start_line INTEGER NOT NULL, start_col INTEGER NOT NULL, end_line INTEGER NOT NULL, end_col INTEGER NOT NULL, start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL, resolved_symbol_id INTEGER REFERENCES symbols(id) ON DELETE SET NULL, confidence REAL NOT NULL, provenance TEXT NOT NULL)`,
`CREATE TABLE refs (id INTEGER PRIMARY KEY, repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, src_symbol_id INTEGER REFERENCES symbols(id) ON DELETE SET NULL, dst_symbol_id INTEGER REFERENCES symbols(id) ON DELETE SET NULL, ref_text TEXT, start_line INTEGER NOT NULL, start_col INTEGER NOT NULL, end_line INTEGER NOT NULL, end_col INTEGER NOT NULL, start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL, confidence REAL NOT NULL, provenance TEXT NOT NULL)`,
}

rawDB, err := sql.Open("sqlite", path)
if err != nil {
t.Fatalf("sql.Open: %v", err)
}
for _, stmt := range v2DDL {
if _, err := rawDB.Exec(stmt); err != nil {
rawDB.Close()
t.Fatalf("v2 DDL %q: %v", stmt[:40], err)
}
}
rawDB.Close()

db, err := kgdb.Open(path)
if err != nil {
t.Fatalf("Open after v2 schema: %v", err)
}
defer db.Close()

var ver int
if err := db.QueryRow("SELECT version FROM schema_version").Scan(&ver); err != nil {
t.Fatalf("read version: %v", err)
}
if ver != 3 {
t.Errorf("expected version=3 after migration, got %d", ver)
}

for _, tbl := range []string{"symbols_fts", "callsites_fts"} {
var cnt int
if err := db.QueryRow(
"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tbl,
).Scan(&cnt); err != nil {
t.Fatalf("check %s: %v", tbl, err)
}
if cnt == 0 {
t.Errorf("expected FTS table %s to exist after migration", tbl)
}
}
}

func TestDeleteFileSymbols(t *testing.T) {
path, cleanup := tmpDB(t)
defer cleanup()

db, _ := kgdb.Open(path)
defer db.Close()

repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")

symbols := []kgdb.SymbolRow{
{
RepoID: repoID, Lang: "go", Kind: "function", Name: "Foo", FQN: "pkg.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1,
},
}
callsites := []kgdb.CallsiteRow{
{
RepoID: repoID, CalleeText: "bar.Bar",
StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10,
Confidence: 0.9, Provenance: "test",
},
}

fileRow := &kgdb.FileRow{RepoID: repoID, Path: "foo.go", Lang: "go", SHA256: "abc", Mtime: 1, Size: 1}
fileID, _, err := kgdb.BatchWrite(db, fileRow, symbols, nil, callsites, nil)
if err != nil {
t.Fatalf("BatchWrite: %v", err)
}

if err := kgdb.DeleteFileSymbols(db, fileID); err != nil {
t.Fatalf("DeleteFileSymbols: %v", err)
}

var symCount, csCount int
_ = db.QueryRow("SELECT COUNT(*) FROM symbols WHERE file_id=?", fileID).Scan(&symCount)
_ = db.QueryRow("SELECT COUNT(*) FROM callsites WHERE file_id=?", fileID).Scan(&csCount)
if symCount != 0 {
t.Errorf("expected 0 symbols after DeleteFileSymbols, got %d", symCount)
}
if csCount != 0 {
t.Errorf("expected 0 callsites after DeleteFileSymbols, got %d", csCount)
}

got, err := kgdb.GetFileByPath(db, repoID, "foo.go")
if err != nil {
t.Fatalf("GetFileByPath: %v", err)
}
if got == nil {
t.Error("expected file row to still exist after DeleteFileSymbols")
}
}

func TestInsertEdgesAndList(t *testing.T) {
path, cleanup := tmpDB(t)
defer cleanup()

db, _ := kgdb.Open(path)
defer db.Close()

repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 1,
})

tx, _ := db.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{
RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "pkg.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1,
},
{
RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Bar", FQN: "pkg.Bar",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1,
},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

tx2, _ := db.Begin()
err = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{
RepoID: repoID, Kind: "CALLS",
SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1],
Confidence: 1.0, Provenance: "test",
},
})
if err != nil {
_ = tx2.Rollback()
t.Fatalf("InsertEdges: %v", err)
}
_ = tx2.Commit()

var count int
_ = db.QueryRow("SELECT COUNT(*) FROM edges WHERE repo_id=?", repoID).Scan(&count)
if count != 1 {
t.Errorf("expected 1 edge, got %d", count)
}
}

func TestListFilesForRepo(t *testing.T) {
path, cleanup := tmpDB(t)
defer cleanup()

db, _ := kgdb.Open(path)
defer db.Close()

repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")

for _, p := range []string{"a.go", "b.go"} {
if _, err := kgdb.UpsertFile(db, &kgdb.FileRow{
RepoID: repoID, Path: p, Lang: "go",
SHA256: "hash-" + p, Mtime: 1, Size: 1,
}); err != nil {
t.Fatalf("UpsertFile %s: %v", p, err)
}
}

files, err := kgdb.ListFilesForRepo(db, repoID)
if err != nil {
t.Fatalf("ListFilesForRepo: %v", err)
}
if len(files) != 2 {
t.Errorf("expected 2 files, got %d", len(files))
}

paths := make(map[string]bool)
for _, f := range files {
paths[f.Path] = true
}
for _, p := range []string{"a.go", "b.go"} {
if !paths[p] {
t.Errorf("expected file %s in result", p)
}
}
}

func TestGetUnresolvedCallsites(t *testing.T) {
path, cleanup := tmpDB(t)
defer cleanup()

db, _ := kgdb.Open(path)
defer db.Close()

repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 1,
})

tx, _ := db.Begin()
symIDs, _ := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{
RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "pkg.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1,
},
})
_ = tx.Commit()

tx2, _ := db.Begin()
err := kgdb.InsertCallsites(tx2, []kgdb.CallsiteRow{
{
RepoID: repoID, FileID: fileID, CalleeText: "alpha",
StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 5,
Confidence: 0.9, Provenance: "test",
},
{
RepoID: repoID, FileID: fileID, CalleeText: "beta",
StartLine: 3, StartCol: 1, EndLine: 3, EndCol: 5,
Confidence: 0.9, Provenance: "test",
},
})
if err != nil {
_ = tx2.Rollback()
t.Fatalf("InsertCallsites: %v", err)
}
_ = tx2.Commit()

var betaID int64
_ = db.QueryRow("SELECT id FROM callsites WHERE callee_text='beta' AND repo_id=?", repoID).Scan(&betaID)
if betaID == 0 {
t.Fatal("couldn't find beta callsite")
}

if err := kgdb.UpdateCallsiteResolved(db, betaID, symIDs[0]); err != nil {
t.Fatalf("UpdateCallsiteResolved: %v", err)
}

unresolved, err := kgdb.GetUnresolvedCallsites(db, repoID, 100)
if err != nil {
t.Fatalf("GetUnresolvedCallsites: %v", err)
}
if len(unresolved) != 1 {
t.Errorf("expected 1 unresolved callsite, got %d", len(unresolved))
}
if len(unresolved) > 0 && unresolved[0].CalleeText != "alpha" {
t.Errorf("expected callee_text 'alpha', got %s", unresolved[0].CalleeText)
}
}

func TestUpdateCallsiteResolved(t *testing.T) {
path, cleanup := tmpDB(t)
defer cleanup()

db, _ := kgdb.Open(path)
defer db.Close()

repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 1,
})

tx, _ := db.Begin()
symIDs, _ := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{
RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "pkg.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1,
},
})
_ = tx.Commit()

tx2, _ := db.Begin()
_ = kgdb.InsertCallsites(tx2, []kgdb.CallsiteRow{
{
RepoID: repoID, FileID: fileID, CalleeText: "someCall",
StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 8,
Confidence: 0.9, Provenance: "test",
},
})
_ = tx2.Commit()

var csID int64
_ = db.QueryRow("SELECT id FROM callsites WHERE repo_id=? AND callee_text='someCall'", repoID).Scan(&csID)
if csID == 0 {
t.Fatal("callsite not found")
}

if err := kgdb.UpdateCallsiteResolved(db, csID, symIDs[0]); err != nil {
t.Fatalf("UpdateCallsiteResolved: %v", err)
}

var resolvedID sql.NullInt64
_ = db.QueryRow("SELECT resolved_symbol_id FROM callsites WHERE id=?", csID).Scan(&resolvedID)
if !resolvedID.Valid || resolvedID.Int64 != symIDs[0] {
t.Errorf("expected resolved_symbol_id=%d, got %v", symIDs[0], resolvedID)
}
}

func TestGenerateStructuralEdges(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")
	fileID, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
		RepoID: repoID, Path: "pkg/foo.go", Lang: "go",
		SHA256: "aaa", Mtime: 1, Size: 1,
	})

	// Write symbols: package, type, concrete method, interface, interface method, field.
	tx, _ := db.Begin()
	symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "package",
			Name: "pkg", FQN: "github.com/example/pkg", StartLine: 1, StartCol: 1, EndLine: 1},
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "type",
			Name: "MyType", FQN: "github.com/example/pkg.MyType", StartLine: 3, StartCol: 1, EndLine: 5},
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "method",
			Name: "DoWork", FQN: "github.com/example/pkg.(MyType).DoWork", StartLine: 7, StartCol: 1, EndLine: 9},
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "interface",
			Name: "MyIface", FQN: "github.com/example/pkg.MyIface", StartLine: 11, StartCol: 1, EndLine: 13},
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "method",
			Name: "Run", FQN: "github.com/example/pkg.MyIface.Run", StartLine: 12, StartCol: 1, EndLine: 12},
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "field",
			Name: "Count", FQN: "github.com/example/pkg.MyType.Count", StartLine: 4, StartCol: 2, EndLine: 4},
		{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
			Name: "NewMyType", FQN: "github.com/example/pkg.NewMyType", StartLine: 15, StartCol: 1, EndLine: 17},
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("InsertSymbols: %v", err)
	}
	_ = tx.Commit()
	_ = symIDs

	if err := kglang.GenerateStructuralEdges(db, repoID); err != nil {
		t.Fatalf("GenerateStructuralEdges: %v", err)
	}

	type edgeKey struct{ kind, src, dst string }
	rows, err := db.Query(`
		SELECT e.kind, s.fqn, d.fqn
		FROM edges e
		JOIN symbols s ON s.id = e.src_symbol_id
		JOIN symbols d ON d.id = e.dst_symbol_id
		WHERE e.repo_id = ?`, repoID)
	if err != nil {
		t.Fatalf("query edges: %v", err)
	}
	defer rows.Close()
	edges := make(map[edgeKey]bool)
	for rows.Next() {
		var k, src, dst string
		_ = rows.Scan(&k, &src, &dst)
		edges[edgeKey{k, src, dst}] = true
	}

	pkg := "github.com/example/pkg"
	want := []edgeKey{
		{"DECLARES", pkg, "github.com/example/pkg.MyType"},
		{"DECLARES", pkg, "github.com/example/pkg.MyIface"},
		{"DECLARES", pkg, "github.com/example/pkg.NewMyType"},
		{"CONTAINS", "github.com/example/pkg.MyType", "github.com/example/pkg.(MyType).DoWork"},
		{"CONTAINS", "github.com/example/pkg.MyIface", "github.com/example/pkg.MyIface.Run"},
		{"CONTAINS", "github.com/example/pkg.MyType", "github.com/example/pkg.MyType.Count"},
		// package → type/interface CONTAINS edges
		{"CONTAINS", pkg, "github.com/example/pkg.MyType"},
		{"CONTAINS", pkg, "github.com/example/pkg.MyIface"},
	}
	for _, w := range want {
		if !edges[w] {
			t.Errorf("missing edge %v; all edges: %v", w, edges)
		}
	}
	// package must NOT CONTAINS functions — design rule: package CONTAINS types only
	if edges[edgeKey{"CONTAINS", pkg, "github.com/example/pkg.NewMyType"}] {
		t.Error("package must not CONTAINS function NewMyType; CONTAINS is for types only")
	}
}

func TestGenerateStructuralEdgesJava(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")
	fileID, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
		RepoID: repoID, Path: "src/com/example/Greeter.java", Lang: "java",
		SHA256: "bbb", Mtime: 1, Size: 1,
	})

	tx, _ := db.Begin()
	symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "package",
			Name: "com.example", FQN: "com.example", StartLine: 1, StartCol: 1, EndLine: 1},
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "class",
			Name: "Greeter", FQN: "com.example.Greeter", StartLine: 3, StartCol: 1, EndLine: 20},
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "constructor",
			Name: "Greeter", FQN: "com.example.Greeter#Greeter(java.lang.String)", StartLine: 6, StartCol: 2, EndLine: 8},
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "method",
			Name: "greet", FQN: "com.example.Greeter#greet():java.lang.String", StartLine: 10, StartCol: 2, EndLine: 12},
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "field",
			Name: "name", FQN: "com.example.Greeter.name", StartLine: 4, StartCol: 2, EndLine: 4},
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "interface",
			Name: "Sayable", FQN: "com.example.Sayable", StartLine: 22, StartCol: 1, EndLine: 25},
		{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "method",
			Name: "say", FQN: "com.example.Sayable#say(java.lang.String):void", StartLine: 23, StartCol: 2, EndLine: 23},
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("InsertSymbols: %v", err)
	}
	_ = tx.Commit()
	_ = symIDs

	if err := kglang.GenerateStructuralEdges(db, repoID); err != nil {
		t.Fatalf("GenerateStructuralEdges: %v", err)
	}

	type edgeKey struct{ kind, src, dst string }
	rows, err := db.Query(`
		SELECT e.kind, s.fqn, d.fqn
		FROM edges e
		JOIN symbols s ON s.id = e.src_symbol_id
		JOIN symbols d ON d.id = e.dst_symbol_id
		WHERE e.repo_id = ?`, repoID)
	if err != nil {
		t.Fatalf("query edges: %v", err)
	}
	defer rows.Close()
	edges := make(map[edgeKey]bool)
	for rows.Next() {
		var k, src, dst string
		_ = rows.Scan(&k, &src, &dst)
		edges[edgeKey{k, src, dst}] = true
	}

	pkg := "com.example"
	cls := "com.example.Greeter"
	iface := "com.example.Sayable"

	want := []edgeKey{
		// package → class/interface CONTAINS
		{"CONTAINS", pkg, cls},
		{"CONTAINS", pkg, iface},
		// class → constructor CONTAINS (via #)
		{"CONTAINS", cls, "com.example.Greeter#Greeter(java.lang.String)"},
		// class → method CONTAINS (via #)
		{"CONTAINS", cls, "com.example.Greeter#greet():java.lang.String"},
		// class → field CONTAINS (via .)
		{"CONTAINS", cls, "com.example.Greeter.name"},
		// interface → method CONTAINS (via #)
		{"CONTAINS", iface, "com.example.Sayable#say(java.lang.String):void"},
	}
	for _, w := range want {
		if !edges[w] {
			t.Errorf("missing Java CONTAINS edge %v; all edges: %v", w, edges)
		}
	}
}

func TestGenerateImportEdges(t *testing.T) {
	path, cleanup := tmpDB(t)
	defer cleanup()

	db, _ := kgdb.Open(path)
	defer db.Close()

	repoID, _ := kgdb.UpsertRepo(db, "/repo", "repo")
	fileA, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
		RepoID: repoID, Path: "a/a.go", Lang: "go",
		SHA256: "aaa", Mtime: 1, Size: 1,
	})
	fileB, _ := kgdb.UpsertFile(db, &kgdb.FileRow{
		RepoID: repoID, Path: "b/b.go", Lang: "go",
		SHA256: "bbb", Mtime: 1, Size: 1,
	})

	tx, _ := db.Begin()
	_, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
		{RepoID: repoID, FileID: fileA, Lang: "go", Kind: "package",
			Name: "a", FQN: "example.com/repo/a", StartLine: 1, StartCol: 1, EndLine: 1},
		{RepoID: repoID, FileID: fileB, Lang: "go", Kind: "package",
			Name: "b", FQN: "example.com/repo/b", StartLine: 1, StartCol: 1, EndLine: 1},
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("InsertSymbols: %v", err)
	}
	_ = tx.Commit()

	// a imports b
	if err := kgdb.StoreImportRefs(db, repoID, fileA, "example.com/repo/a", []string{"example.com/repo/b"}); err != nil {
		t.Fatalf("StoreImportRefs: %v", err)
	}

	if err := kglang.GenerateImportEdges(db, repoID); err != nil {
		t.Fatalf("GenerateImportEdges: %v", err)
	}

	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN symbols s ON s.id = e.src_symbol_id
		JOIN symbols d ON d.id = e.dst_symbol_id
		WHERE e.repo_id=? AND e.kind='IMPORTS'
		  AND s.fqn='example.com/repo/a' AND d.fqn='example.com/repo/b'`,
		repoID).Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 IMPORTS edge, got %d", count)
	}

	// Idempotent: running again should not duplicate edges.
	_ = kglang.GenerateImportEdges(db, repoID)
	_ = db.QueryRow("SELECT COUNT(*) FROM edges WHERE repo_id=? AND kind='IMPORTS'", repoID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 IMPORTS edge after second run, got %d", count)
	}
}
