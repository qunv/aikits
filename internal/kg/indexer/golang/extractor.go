package golang

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"aikits/internal/kg/db"
)

// ExtractGoSymbols extracts symbols and callsites from a parsed Go AST.
// modulePath is the Go module path (e.g. "aikits").
// pkgPath is the module-relative package path (e.g. "internal/command").
func ExtractGoSymbols(
	file *ast.File,
	fset *token.FileSet,
	repoID, fileID int64,
	modulePath, pkgPath string,
) ([]db.SymbolRow, []db.CallsiteRow) {
	var symbols []db.SymbolRow
	var callsites []db.CallsiteRow

	pkgName := file.Name.Name
	fqnBase := modulePath
	if pkgPath != "" {
		fqnBase = modulePath + "/" + pkgPath
	}

	// Package-level symbol
	pkgPos := fset.Position(file.Package)
	symbols = append(symbols, db.SymbolRow{
		RepoID:    repoID,
		FileID:    fileID,
		Lang:      "go",
		Kind:      "package",
		Name:      pkgName,
		FQN:       fqnBase,
		StartLine: pkgPos.Line,
		StartCol:  pkgPos.Column,
		EndLine:   pkgPos.Line,
		EndCol:    pkgPos.Column,
		StartByte: fset.File(file.Package).Offset(file.Package),
		EndByte:   fset.File(file.Package).Offset(file.Package),
		Visibility: "public",
	})

	// Build a map from AST node position ranges to symbol index for caller resolution
	type funcRange struct {
		start, end token.Pos
		idx        int
	}
	var funcRanges []funcRange

	// Extract top-level declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := extractFuncDecl(d, fset, repoID, fileID, fqnBase)
			idx := len(symbols)
			symbols = append(symbols, sym)
			if d.Body != nil {
				funcRanges = append(funcRanges, funcRange{
					start: d.Body.Lbrace,
					end:   d.Body.Rbrace,
					idx:   idx,
				})
			}

		case *ast.GenDecl:
			syms := extractGenDecl(d, fset, repoID, fileID, fqnBase, pkgName)
			symbols = append(symbols, syms...)
		}
	}

	// Extract callsites by walking all call expressions
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		calleeText := exprText(call.Fun)
		if calleeText == "" {
			return true
		}

		startPos := fset.Position(call.Pos())
		endPos := fset.Position(call.End())

		var callerIdx *int
		for _, fr := range funcRanges {
			if call.Pos() >= fr.start && call.End() <= fr.end {
				i := fr.idx
				callerIdx = &i
				// pick smallest enclosing range
				break
			}
		}

		var callerSymID *int64
		if callerIdx != nil && *callerIdx < len(symbols) {
			// We'll fill in real IDs after DB insert; for now use placeholder 0
			// The command layer will fix this after InsertSymbols
			_ = callerIdx
		}

		callsites = append(callsites, db.CallsiteRow{
			RepoID:         repoID,
			FileID:         fileID,
			CallerSymbolID: callerSymID,
			CalleeText:     calleeText,
			StartLine:      startPos.Line,
			StartCol:       startPos.Column,
			EndLine:        endPos.Line,
			EndCol:         endPos.Column,
			StartByte:      startPos.Offset,
			EndByte:        endPos.Offset,
			Confidence:     0.9,
			Provenance:     "go/ast",
		})
		return true
	})

	return symbols, callsites
}

func extractFuncDecl(d *ast.FuncDecl, fset *token.FileSet, repoID, fileID int64, fqnBase string) db.SymbolRow {
	name := d.Name.Name
	kind := "function"
	fqn := fqnBase + "." + name

	var sig strings.Builder
	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = "method"
		recv := d.Recv.List[0]
		recvType := exprText(recv.Type)
		sig.WriteString("(")
		sig.WriteString(recvType)
		sig.WriteString(") ")
		// Include receiver type in FQN
		cleanRecv := strings.TrimPrefix(strings.TrimPrefix(recvType, "*"), "(")
		cleanRecv = strings.TrimSuffix(cleanRecv, ")")
		fqn = fqnBase + ".(" + cleanRecv + ")." + name
	}
	sig.WriteString(name)
	if d.Type != nil {
		sig.WriteString(formatFuncType(d.Type, fset))
	}

	visibility := "private"
	if ast.IsExported(name) {
		visibility = "public"
	}

	startPos := fset.Position(d.Pos())
	endPos := fset.Position(d.End())

	doc := ""
	if d.Doc != nil {
		doc = d.Doc.Text()
	}

	return db.SymbolRow{
		RepoID:     repoID,
		FileID:     fileID,
		Lang:       "go",
		Kind:       kind,
		Name:       name,
		FQN:        fqn,
		Signature:  sig.String(),
		Visibility: visibility,
		Doc:        doc,
		StartLine:  startPos.Line,
		StartCol:   startPos.Column,
		EndLine:    endPos.Line,
		EndCol:     endPos.Column,
		StartByte:  startPos.Offset,
		EndByte:    endPos.Offset,
	}
}

func extractGenDecl(d *ast.GenDecl, fset *token.FileSet, repoID, fileID int64, fqnBase, pkgName string) []db.SymbolRow {
	var syms []db.SymbolRow

	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			kind := "type"
			switch s.Type.(type) {
			case *ast.InterfaceType:
				kind = "interface"
			case *ast.StructType:
				kind = "type"
			}
			name := s.Name.Name
			fqn := fqnBase + "." + name
			visibility := "private"
			if ast.IsExported(name) {
				visibility = "public"
			}
			startPos := fset.Position(s.Pos())
			endPos := fset.Position(s.End())
			doc := ""
			if s.Comment != nil {
				doc = s.Comment.Text()
			} else if d.Doc != nil {
				doc = d.Doc.Text()
			}
			syms = append(syms, db.SymbolRow{
				RepoID:     repoID,
				FileID:     fileID,
				Lang:       "go",
				Kind:       kind,
				Name:       name,
				FQN:        fqn,
				Visibility: visibility,
				Doc:        doc,
				StartLine:  startPos.Line,
				StartCol:   startPos.Column,
				EndLine:    endPos.Line,
				EndCol:     endPos.Column,
				StartByte:  startPos.Offset,
				EndByte:    endPos.Offset,
			})

			// Extract methods from interfaces
			if iface, ok := s.Type.(*ast.InterfaceType); ok && iface.Methods != nil {
				for _, method := range iface.Methods.List {
					for _, mname := range method.Names {
						mFQN := fqn + "." + mname.Name
						mvis := "private"
						if ast.IsExported(mname.Name) {
							mvis = "public"
						}
						mStart := fset.Position(method.Pos())
						mEnd := fset.Position(method.End())
						syms = append(syms, db.SymbolRow{
							RepoID:     repoID,
							FileID:     fileID,
							Lang:       "go",
							Kind:       "method",
							Name:       mname.Name,
							FQN:        mFQN,
							Visibility: mvis,
							StartLine:  mStart.Line,
							StartCol:   mStart.Column,
							EndLine:    mEnd.Line,
							EndCol:     mEnd.Column,
							StartByte:  mStart.Offset,
							EndByte:    mEnd.Offset,
						})
					}
				}
			}

			// Extract fields from structs
			if st, ok := s.Type.(*ast.StructType); ok && st.Fields != nil {
				for _, field := range st.Fields.List {
					for _, fname := range field.Names {
						fFQN := fqn + "." + fname.Name
						fvis := "private"
						if ast.IsExported(fname.Name) {
							fvis = "public"
						}
						fStart := fset.Position(field.Pos())
						fEnd := fset.Position(field.End())
						syms = append(syms, db.SymbolRow{
							RepoID:     repoID,
							FileID:     fileID,
							Lang:       "go",
							Kind:       "field",
							Name:       fname.Name,
							FQN:        fFQN,
							Visibility: fvis,
							StartLine:  fStart.Line,
							StartCol:   fStart.Column,
							EndLine:    fEnd.Line,
							EndCol:     fEnd.Column,
							StartByte:  fStart.Offset,
							EndByte:    fEnd.Offset,
						})
					}
				}
			}

		case *ast.ValueSpec:
			kind := "var"
			switch d.Tok.String() {
			case "const":
				kind = "const"
			case "var":
				kind = "var"
			}
			for _, name := range s.Names {
				fqn := fqnBase + "." + name.Name
				visibility := "private"
				if ast.IsExported(name.Name) {
					visibility = "public"
				}
				startPos := fset.Position(s.Pos())
				endPos := fset.Position(s.End())
				doc := ""
				if s.Comment != nil {
					doc = s.Comment.Text()
				} else if d.Doc != nil {
					doc = d.Doc.Text()
				}
				syms = append(syms, db.SymbolRow{
					RepoID:     repoID,
					FileID:     fileID,
					Lang:       "go",
					Kind:       kind,
					Name:       name.Name,
					FQN:        fqn,
					Visibility: visibility,
					Doc:        doc,
					StartLine:  startPos.Line,
					StartCol:   startPos.Column,
					EndLine:    endPos.Line,
					EndCol:     endPos.Column,
					StartByte:  startPos.Offset,
					EndByte:    endPos.Offset,
				})
			}

		case *ast.ImportSpec:
			_ = pkgName
		}
	}
	return syms
}

// exprText returns a textual representation of an expression.
func exprText(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprText(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprText(e.X)
	case *ast.IndexExpr:
		return exprText(e.X) + "[" + exprText(e.Index) + "]"
	case *ast.ArrayType:
		return "[]" + exprText(e.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprText(e.Key), exprText(e.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.ParenExpr:
		return "(" + exprText(e.X) + ")"
	default:
		return ""
	}
}

// goBuiltinNames is the set of Go predeclared type identifiers that should not
// generate REFERENCES edges.
var goBuiltinNames = map[string]bool{
	"bool": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
	"string": true, "byte": true, "rune": true, "error": true,
	"any": true, "comparable": true,
}

// ExtractGoTypeRefs extracts non-call type references from a parsed Go AST.
// Produces TypeRef pairs (srcFQN, typeFQN) for:
//   - function/method parameter and return types
//   - struct field types
//   - interface method parameter and return types
//
// Cross-package references are resolved via import paths (alias → importPath).
// Same-package references use fqnBase + "." + typeName as the resolved FQN.
// SelectorExpr whose package alias is not in the import map (e.g. dot imports)
// are skipped to avoid false positives.
func ExtractGoTypeRefs(file *ast.File, modPath, pkgPath string) []db.TypeRef {
	fqnBase := modPath
	if pkgPath != "" {
		fqnBase = modPath + "/" + pkgPath
	}

	// Build package alias → import path map
	pkgMap := make(map[string]string)
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		importPath := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			if imp.Name.Name == "." || imp.Name.Name == "_" {
				continue
			}
			pkgMap[imp.Name.Name] = importPath
		} else {
			parts := strings.Split(importPath, "/")
			pkgMap[parts[len(parts)-1]] = importPath
		}
	}

	seen := make(map[[2]string]bool)
	var refs []db.TypeRef

	addRef := func(srcFQN, typeRefFQN string) {
		if srcFQN == typeRefFQN {
			return
		}
		k := [2]string{srcFQN, typeRefFQN}
		if !seen[k] {
			seen[k] = true
			refs = append(refs, db.TypeRef{SrcFQN: srcFQN, TypeName: typeRefFQN})
		}
	}

	var collectTypeRefs func(expr ast.Expr, srcFQN string)
	collectTypeRefs = func(expr ast.Expr, srcFQN string) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.Ident:
			if !goBuiltinNames[e.Name] {
				addRef(srcFQN, fqnBase+"."+e.Name)
			}
		case *ast.SelectorExpr:
			if x, ok := e.X.(*ast.Ident); ok {
				if importPath, found := pkgMap[x.Name]; found {
					addRef(srcFQN, importPath+"."+e.Sel.Name)
				}
				// Unresolved alias (dot imports, generated code) → skip
			}
		case *ast.StarExpr:
			collectTypeRefs(e.X, srcFQN)
		case *ast.ArrayType:
			collectTypeRefs(e.Elt, srcFQN)
		case *ast.MapType:
			collectTypeRefs(e.Key, srcFQN)
			collectTypeRefs(e.Value, srcFQN)
		case *ast.ChanType:
			collectTypeRefs(e.Value, srcFQN)
		case *ast.Ellipsis:
			collectTypeRefs(e.Elt, srcFQN)
		case *ast.IndexExpr:
			// Generic with single type param: T[A]
			collectTypeRefs(e.X, srcFQN)
			collectTypeRefs(e.Index, srcFQN)
		case *ast.IndexListExpr:
			// Generic with multiple type params: T[A, B]
			collectTypeRefs(e.X, srcFQN)
			for _, idx := range e.Indices {
				collectTypeRefs(idx, srcFQN)
			}
		}
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			funcFQN := fqnBase + "." + name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv := d.Recv.List[0]
				recvType := exprText(recv.Type)
				cleanRecv := strings.TrimPrefix(strings.TrimPrefix(recvType, "*"), "(")
				cleanRecv = strings.TrimSuffix(cleanRecv, ")")
				funcFQN = fqnBase + ".(" + cleanRecv + ")." + name
			}
			if d.Type.Params != nil {
				for _, p := range d.Type.Params.List {
					collectTypeRefs(p.Type, funcFQN)
				}
			}
			if d.Type.Results != nil {
				for _, r := range d.Type.Results.List {
					collectTypeRefs(r.Type, funcFQN)
				}
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				typeFQN := fqnBase + "." + ts.Name.Name
				switch t := ts.Type.(type) {
				case *ast.StructType:
					if t.Fields != nil {
						for _, field := range t.Fields.List {
							collectTypeRefs(field.Type, typeFQN)
						}
					}
				case *ast.InterfaceType:
					if t.Methods != nil {
						for _, m := range t.Methods.List {
							if len(m.Names) == 0 {
								// Embedded interface — associate with parent type
								collectTypeRefs(m.Type, typeFQN)
								continue
							}
							methodFQN := typeFQN + "." + m.Names[0].Name
							if ft, ok := m.Type.(*ast.FuncType); ok {
								if ft.Params != nil {
									for _, p := range ft.Params.List {
										collectTypeRefs(p.Type, methodFQN)
									}
								}
								if ft.Results != nil {
									for _, r := range ft.Results.List {
										collectTypeRefs(r.Type, methodFQN)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return refs
}

// ExtractGoImports returns the import paths declared in a Go source file.
// Each path is the unquoted import path (e.g. "fmt", "aikits/internal/kg/db").
func ExtractGoImports(file *ast.File) []string {
	var paths []string
	for _, imp := range file.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			if path != "" {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

func formatFuncType(ft *ast.FuncType, fset *token.FileSet) string {
	_ = fset
	var sb strings.Builder
	sb.WriteString("(")
	if ft.Params != nil {
		for i, p := range ft.Params.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			if len(p.Names) > 0 {
				for j, n := range p.Names {
					if j > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(n.Name)
				}
				sb.WriteString(" ")
			}
			sb.WriteString(exprText(p.Type))
		}
	}
	sb.WriteString(")")
	if ft.Results != nil && len(ft.Results.List) > 0 {
		sb.WriteString(" ")
		if len(ft.Results.List) > 1 {
			sb.WriteString("(")
		}
		for i, r := range ft.Results.List {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(exprText(r.Type))
		}
		if len(ft.Results.List) > 1 {
			sb.WriteString(")")
		}
	}
	return sb.String()
}
