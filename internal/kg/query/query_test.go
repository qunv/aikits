package query_test

import (
	"path/filepath"
	"testing"

	kgdb "aikits/internal/kg/db"
	"aikits/internal/kg/query"
)

func setupTestDB(t *testing.T) (*kgdb.RepoRow, interface{ Close() error }) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")

	db, err := kgdb.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}

	repoID, err := kgdb.UpsertRepo(db, "/repo", "testrepo")
	if err != nil {
		t.Fatalf("UpsertRepo: %v", err)
	}

	fileID, err := kgdb.UpsertFile(db, &kgdb.FileRow{
		RepoID: repoID, Path: "main.go", Lang: "go",
		SHA256: "aaa", Mtime: 1, Size: 100,
	})
	if err != nil {
		t.Fatalf("UpsertFile: %v", err)
	}

	syms := []kgdb.SymbolRow{
		{
			RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
			Name: "Foo", FQN: "github.com/example.Foo",
			StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1,
		},
		{
			RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
			Name: "Bar", FQN: "github.com/example.Bar",
			StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1,
		},
		{
			RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
			Name: "Baz", FQN: "github.com/example.Baz",
			StartLine: 12, StartCol: 1, EndLine: 15, EndCol: 1,
		},
	}

	tx, _ := db.Begin()
	symIDs, err := kgdb.InsertSymbols(tx, syms)
	if err != nil {
		tx.Rollback()
		t.Fatalf("InsertSymbols: %v", err)
	}
	tx.Commit()

	// Foo calls Bar, Bar calls Baz
	if len(symIDs) >= 3 && symIDs[0] != 0 && symIDs[1] != 0 && symIDs[2] != 0 {
		tx2, _ := db.Begin()
		_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
			{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1], Confidence: 1.0, Provenance: "test"},
			{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[1], DstSymbolID: symIDs[2], Confidence: 1.0, Provenance: "test"},
		})
		tx2.Commit()
	}

	repo, _ := kgdb.GetRepoByPath(db, "/repo")
	return repo, db
}

func TestLookupByFQN(t *testing.T) {
	repo, dbCloser := setupTestDB(t)
	_ = repo
	_ = dbCloser
	// covered by TestLookupFunctions
}

func TestLookupFunctions(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")

	sqlDB, err := kgdb.Open(dbPath)
	if err != nil {
		t.Fatalf("Open DB: %v", err)
	}
	defer sqlDB.Close()

	repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "testrepo")
	fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
		RepoID: repoID, Path: "main.go", Lang: "go",
		SHA256: "aaa", Mtime: 1, Size: 100,
	})

	syms := []kgdb.SymbolRow{
		{
			RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
			Name: "Foo", FQN: "github.com/example.Foo",
			StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1,
		},
		{
			RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
			Name: "Bar", FQN: "github.com/example.Bar",
			StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1,
		},
	}

	tx, _ := sqlDB.Begin()
	symIDs, err := kgdb.InsertSymbols(tx, syms)
	if err != nil {
		tx.Rollback()
		t.Fatalf("InsertSymbols: %v", err)
	}
	_ = tx.Commit()

	// LookupByFQN
	results, err := query.LookupByFQN(sqlDB, repoID, "github.com/example.Foo")
	if err != nil {
		t.Fatalf("LookupByFQN: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("LookupByFQN: want 1 result, got %d", len(results))
	}

	// LookupByName partial
	results2, err := query.LookupByName(sqlDB, repoID, "Ba")
	if err != nil {
		t.Fatalf("LookupByName: %v", err)
	}
	if len(results2) != 1 {
		t.Errorf("LookupByName 'Ba': want 1 result, got %d", len(results2))
	}

	// BFS over CALLS edges
	if len(symIDs) >= 2 {
		tx2, _ := sqlDB.Begin()
		_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
			{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1], Confidence: 1.0, Provenance: "test"},
		})
		_ = tx2.Commit()

		// Foo -> Bar: callees of Foo should include Bar
		ids, bfsErr := query.BFS(sqlDB, repoID, symIDs[0], 2, 10, query.Outbound, []string{"CALLS"})
		if bfsErr != nil {
			t.Fatalf("BFS: %v", bfsErr)
		}
		if len(ids) != 1 || ids[0].ID != symIDs[1] {
			t.Errorf("BFS callees of Foo: want [%d], got %v", symIDs[1], ids)
		}

		// Callers of Bar should include Foo
		ids2, bfsErr2 := query.BFS(sqlDB, repoID, symIDs[1], 2, 10, query.Inbound, []string{"CALLS"})
		if bfsErr2 != nil {
			t.Fatalf("BFS inbound: %v", bfsErr2)
		}
		if len(ids2) != 1 || ids2[0].ID != symIDs[0] {
			t.Errorf("BFS callers of Bar: want [%d], got %v", symIDs[0], ids2)
		}
	}
}

// ─── Callers / Callees ───────────────────────────────────────────────────────

func TestCallers(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "github.com/example.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Bar", FQN: "github.com/example.Bar",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

tx2, _ := sqlDB.Begin()
_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1], Confidence: 1.0, Provenance: "test"},
})
_ = tx2.Commit()

nodes, err := query.Callers(sqlDB, repoID, "github.com/example.Bar", 2, 10)
if err != nil {
t.Fatalf("Callers: %v", err)
}
if len(nodes) != 1 {
t.Errorf("expected 1 caller of Bar, got %d", len(nodes))
}
if len(nodes) > 0 && nodes[0].Symbol.Name != "Foo" {
t.Errorf("expected caller=Foo, got %s", nodes[0].Symbol.Name)
}
}

func TestCallees(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "github.com/example.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Bar", FQN: "github.com/example.Bar",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

tx2, _ := sqlDB.Begin()
_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1], Confidence: 1.0, Provenance: "test"},
})
_ = tx2.Commit()

nodes, err := query.Callees(sqlDB, repoID, "github.com/example.Foo", 2, 10)
if err != nil {
t.Fatalf("Callees: %v", err)
}
if len(nodes) != 1 {
t.Errorf("expected 1 callee of Foo, got %d", len(nodes))
}
if len(nodes) > 0 && nodes[0].Symbol.Name != "Bar" {
t.Errorf("expected callee=Bar, got %s", nodes[0].Symbol.Name)
}
}

func TestImpls(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "interface",
Name: "IFace", FQN: "github.com/example.IFace",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "type",
Name: "Impl", FQN: "github.com/example.Impl",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

// Impl IMPLEMENTS IFace: src=Impl, dst=IFace
tx2, _ := sqlDB.Begin()
_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{RepoID: repoID, Kind: "IMPLEMENTS", SrcSymbolID: symIDs[1], DstSymbolID: symIDs[0], Confidence: 1.0, Provenance: "test"},
})
_ = tx2.Commit()

nodes, err := query.Impls(sqlDB, repoID, "github.com/example.IFace", 2, 10)
if err != nil {
t.Fatalf("Impls: %v", err)
}
if len(nodes) != 1 {
t.Errorf("expected 1 impl of IFace, got %d", len(nodes))
}
if len(nodes) > 0 && nodes[0].Symbol.Name != "Impl" {
t.Errorf("expected Impl, got %s", nodes[0].Symbol.Name)
}
}

func TestOverrides(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "method",
Name: "BaseM", FQN: "com.example.Base#run():void",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "java", Kind: "method",
Name: "SubM", FQN: "com.example.Sub#run():void",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

// SubM OVERRIDES BaseM: src=SubM, dst=BaseM
tx2, _ := sqlDB.Begin()
_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{RepoID: repoID, Kind: "OVERRIDES", SrcSymbolID: symIDs[1], DstSymbolID: symIDs[0], Confidence: 1.0, Provenance: "test"},
})
_ = tx2.Commit()

nodes, err := query.Overrides(sqlDB, repoID, "com.example.Base#run():void", 2, 10)
if err != nil {
t.Fatalf("Overrides: %v", err)
}
if len(nodes) != 1 {
t.Errorf("expected 1 override of BaseM, got %d", len(nodes))
}
if len(nodes) > 0 && nodes[0].Symbol.Name != "SubM" {
t.Errorf("expected SubM, got %s", nodes[0].Symbol.Name)
}
}

func TestImpact(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "github.com/example.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Bar", FQN: "github.com/example.Bar",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Baz", FQN: "github.com/example.Baz",
StartLine: 12, StartCol: 1, EndLine: 15, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

// Foo CALLS Bar, Baz REFERENCES Bar
tx2, _ := sqlDB.Begin()
_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1], Confidence: 1.0, Provenance: "test"},
{RepoID: repoID, Kind: "REFERENCES", SrcSymbolID: symIDs[2], DstSymbolID: symIDs[1], Confidence: 1.0, Provenance: "test"},
})
_ = tx2.Commit()

nodes, err := query.Impact(sqlDB, repoID, "github.com/example.Bar", 2, 10)
if err != nil {
t.Fatalf("Impact: %v", err)
}
if len(nodes) != 2 {
t.Errorf("expected 2 impacted symbols (Foo+Baz), got %d", len(nodes))
for _, n := range nodes {
t.Logf("  node: %s", n.Symbol.Name)
}
}

names := make(map[string]bool)
for _, n := range nodes {
names[n.Symbol.Name] = true
}
if !names["Foo"] {
t.Error("expected Foo in impact results")
}
if !names["Baz"] {
t.Error("expected Baz in impact results")
}
}

func TestIterateSymbols(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})
fileID2, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "Main.java", Lang: "java", SHA256: "bbb", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
_, err = kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "github.com/example.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Bar", FQN: "github.com/example.Bar",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
{RepoID: repoID, FileID: fileID2, Lang: "java", Kind: "class",
Name: "Main", FQN: "com.example.Main",
StartLine: 1, StartCol: 1, EndLine: 20, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

var allCount int
if err := query.IterateSymbols(sqlDB, repoID, "", func(s query.Symbol) error {
allCount++
return nil
}); err != nil {
t.Fatalf("IterateSymbols all: %v", err)
}
if allCount != 3 {
t.Errorf("expected 3 symbols, got %d", allCount)
}

var goCount int
if err := query.IterateSymbols(sqlDB, repoID, "go", func(s query.Symbol) error {
goCount++
return nil
}); err != nil {
t.Fatalf("IterateSymbols go: %v", err)
}
if goCount != 2 {
t.Errorf("expected 2 go symbols, got %d", goCount)
}
}

func TestIterateEdges(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Foo", FQN: "github.com/example.Foo",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Bar", FQN: "github.com/example.Bar",
StartLine: 7, StartCol: 1, EndLine: 10, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

tx2, _ := sqlDB.Begin()
_ = kgdb.InsertEdges(tx2, []kgdb.EdgeRow{
{RepoID: repoID, Kind: "CALLS", SrcSymbolID: symIDs[0], DstSymbolID: symIDs[1], Confidence: 0.9, Provenance: "test"},
})
_ = tx2.Commit()

var edges []query.EdgeResult
if err := query.IterateEdges(sqlDB, repoID, func(e query.EdgeResult) error {
edges = append(edges, e)
return nil
}); err != nil {
t.Fatalf("IterateEdges: %v", err)
}
if len(edges) == 0 {
t.Fatal("expected at least 1 edge")
}
e := edges[0]
if e.Kind != "CALLS" {
t.Errorf("expected kind=CALLS, got %s", e.Kind)
}
if e.SrcSymbolID != symIDs[0] || e.DstSymbolID != symIDs[1] {
t.Errorf("edge src/dst mismatch: got src=%d dst=%d", e.SrcSymbolID, e.DstSymbolID)
}
}

func TestGetSymbolByID(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "MyFunc", FQN: "github.com/example.MyFunc",
StartLine: 10, StartCol: 1, EndLine: 20, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

sym, err := query.GetSymbolByID(sqlDB, symIDs[0])
if err != nil {
t.Fatalf("GetSymbolByID: %v", err)
}
if sym.Name != "MyFunc" {
t.Errorf("expected Name=MyFunc, got %s", sym.Name)
}
if sym.FQN != "github.com/example.MyFunc" {
t.Errorf("expected FQN=github.com/example.MyFunc, got %s", sym.FQN)
}
if sym.Lang != "go" {
t.Errorf("expected Lang=go, got %s", sym.Lang)
}
}

func TestGetSymbolIDByFQN(t *testing.T) {
dir := t.TempDir()
sqlDB, err := kgdb.Open(filepath.Join(dir, "test.sqlite"))
if err != nil {
t.Fatalf("Open: %v", err)
}
defer sqlDB.Close()

repoID, _ := kgdb.UpsertRepo(sqlDB, "/repo", "repo")
fileID, _ := kgdb.UpsertFile(sqlDB, &kgdb.FileRow{
RepoID: repoID, Path: "main.go", Lang: "go", SHA256: "aaa", Mtime: 1, Size: 100,
})

tx, _ := sqlDB.Begin()
symIDs, err := kgdb.InsertSymbols(tx, []kgdb.SymbolRow{
{RepoID: repoID, FileID: fileID, Lang: "go", Kind: "function",
Name: "Hello", FQN: "github.com/example.Hello",
StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1},
})
if err != nil {
_ = tx.Rollback()
t.Fatalf("InsertSymbols: %v", err)
}
_ = tx.Commit()

id, err := query.GetSymbolIDByFQN(sqlDB, repoID, "github.com/example.Hello")
if err != nil {
t.Fatalf("GetSymbolIDByFQN: %v", err)
}
if id != symIDs[0] {
t.Errorf("expected ID=%d, got %d", symIDs[0], id)
}

// Missing FQN returns 0
id2, err := query.GetSymbolIDByFQN(sqlDB, repoID, "github.com/example.NoExist")
if err != nil {
t.Fatalf("GetSymbolIDByFQN missing: %v", err)
}
if id2 != 0 {
t.Errorf("expected 0 for missing FQN, got %d", id2)
}
}
