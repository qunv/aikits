package kg

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"go.uber.org/zap"

	kgdb "aikits/internal/kg/db"
	"aikits/internal/kg/indexer"
	kglang "aikits/internal/kg/lang"
	"aikits/internal/kg/pathutil"
)

type indexResult struct {
	fileRow   *kgdb.FileRow
	idxer     kglang.Indexer
	ext       kglang.FileExtract
	unchanged bool
	err       error
	relPath   string
}

// Index scans the repository source files and updates the knowledge graph.
func (kg *KG) Index(_ context.Context, opts IndexOptions) (*IndexResult, error) {
	if opts.Full {
		if err := kgdb.ClearRepoData(kg.db, kg.repo.ID); err != nil {
			return nil, fmt.Errorf("clear repo data for full re-index: %w", err)
		}
		kg.log.Info("cleared existing index for full re-index")
	}

	langFlag := langsToFlag(opts.Lang)
	langs := kglang.ParseLangs(langFlag)

	walker := indexer.NewWalker(kg.root, langs)
	files, err := walker.Walk()
	if err != nil {
		return nil, fmt.Errorf("file discovery: %w", err)
	}

	dbFiles, err := kgdb.ListFilesForRepo(kg.db, kg.repo.ID)
	if err != nil {
		return nil, fmt.Errorf("list db files: %w", err)
	}
	dbFileMap := make(map[string]*kgdb.FileRow, len(dbFiles))
	for i := range dbFiles {
		dbFileMap[dbFiles[i].Path] = &dbFiles[i]
	}

	onDisk := make(map[string]bool, len(files))
	for _, f := range files {
		onDisk[f.RelPath] = true
	}
	for _, df := range dbFiles {
		if !onDisk[df.Path] {
			if delErr := kgdb.DeleteFile(kg.db, df.ID); delErr != nil {
				kg.log.Warn("delete stale file", zap.String("path", df.Path), zap.Error(delErr))
			}
		}
	}

	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}

	langIndexers := map[string]kglang.Indexer{
		"go":         kglang.NewGoIndexer(kg.root),
		"java":       &kglang.JavaIndexer{},
		"javascript": &kglang.JavaScriptIndexer{},
		"html":       &kglang.HTMLIndexer{},
		"css":        &kglang.CSSIndexer{},
	}

	sem := make(chan struct{}, jobs)
	results := make([]indexResult, len(files))
	var wg sync.WaitGroup

	for i, f := range files {
		wg.Add(1)
		go func(idx int, df indexer.DiscoveredFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := indexResult{relPath: df.RelPath}

			info, statErr := os.Stat(df.AbsPath)
			if statErr != nil {
				res.err = statErr
				results[idx] = res
				return
			}

			sha, shaErr := indexer.ComputeSHA256(df.AbsPath)
			if shaErr != nil {
				res.err = shaErr
				results[idx] = res
				return
			}

			dbFile := dbFileMap[df.RelPath]
			if !opts.Full && !indexer.FileChanged(dbFile, df.AbsPath, info, sha) {
				res.unchanged = true
				results[idx] = res
				return
			}

			src, readErr := os.ReadFile(df.AbsPath)
			if readErr != nil {
				res.err = readErr
				results[idx] = res
				return
			}

			idxer, ok := langIndexers[df.Lang]
			if !ok {
				results[idx] = res
				return
			}

			ext, extractErr := idxer.Extract(src, df.AbsPath, df.RelPath, kg.repo.ID)
			if extractErr != nil {
				kg.log.Warn("parse error", zap.String("file", df.RelPath), zap.Error(extractErr))
			}

			res.fileRow = &kgdb.FileRow{
				RepoID: kg.repo.ID,
				Path:   pathutil.ToSlash(df.RelPath),
				Lang:   df.Lang,
				SHA256: sha,
				Mtime:  info.ModTime().Unix(),
				Size:   info.Size(),
			}
			res.idxer = idxer
			res.ext = ext
			results[idx] = res
		}(i, f)
	}
	wg.Wait()

	callsInserters := kglang.DefaultCallsEdgeInserters()
	out := &IndexResult{}
	for _, r := range results {
		if r.unchanged {
			out.Unchanged++
			continue
		}
		if r.err != nil {
			kg.log.Warn("index error", zap.String("file", r.relPath), zap.Error(r.err))
			out.Errors++
			continue
		}
		if r.fileRow == nil {
			continue
		}
		fileID, _, batchErr := kgdb.BatchWrite(kg.db, r.fileRow, r.ext.Symbols, nil, r.ext.Callsites, callsInserters)
		if batchErr != nil {
			kg.log.Warn("batch write error", zap.String("file", r.relPath), zap.Error(batchErr))
			out.Errors++
			continue
		}
		if r.idxer != nil {
			if refErr := r.idxer.StoreRefs(kg.db, kg.repo.ID, fileID, r.ext); refErr != nil {
				kg.log.Warn("store refs error", zap.String("file", r.relPath), zap.Error(refErr))
			}
		}
		out.Indexed++
		out.Symbols += len(r.ext.Symbols)
		out.Callsites += len(r.ext.Callsites)
	}

	if err := kglang.GenerateStructuralEdges(kg.db, kg.repo.ID); err != nil {
		kg.log.Warn("generate structural edges", zap.Error(err))
	}
	if err := kglang.GenerateImportEdges(kg.db, kg.repo.ID); err != nil {
		kg.log.Warn("generate import edges", zap.Error(err))
	}
	if err := kglang.GenerateExtendsEdges(kg.db, kg.repo.ID); err != nil {
		kg.log.Warn("generate extends edges", zap.Error(err))
	}
	if err := kglang.GenerateOverridesEdges(kg.db, kg.repo.ID); err != nil {
		kg.log.Warn("generate overrides edges", zap.Error(err))
	}
	if err := kglang.GenerateReferencesEdges(kg.db, kg.repo.ID); err != nil {
		kg.log.Warn("generate references edges", zap.Error(err))
	}

	return out, nil
}

// langsToFlag converts a []Lang into the comma-separated string that
// internal/kg/lang.ParseLangs understands.
func langsToFlag(langs []Lang) string {
	if len(langs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(langs))
	for _, l := range langs {
		if l != LangAll {
			parts = append(parts, string(l))
		}
	}
	return strings.Join(parts, ",")
}
