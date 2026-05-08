package java

import kgdb "aikits/internal/kg/db"

// ExtractJavaSymbols extracts symbols and callsites from Java source using
// tree-sitter AST parsing. All symbols get confidence=0.5, provenance="heuristic".
func ExtractJavaSymbols(src []byte, repoID, fileID int64) ([]kgdb.SymbolRow, []kgdb.CallsiteRow) {
r := ExtractJava(src, repoID, fileID)
return r.Symbols, r.Callsites
}

// ExtractJavaImports returns the imported package FQNs from Java source.
// For "import com.example.Foo;" it returns "com.example".
// For "import com.example.*;" it returns "com.example".
// Static imports are skipped.
func ExtractJavaImports(src []byte) []string {
return ExtractJava(src, 0, 0).ImportPaths
}

// ExtractJavaExtendsRefs parses Java source to find explicit extends declarations.
// Returns one ExtendsRef per class/interface that has an extends clause.
func ExtractJavaExtendsRefs(src []byte) []kgdb.ExtendsRef {
return ExtractJava(src, 0, 0).ExtendsRefs
}

// ExtractJavaImplementsRefs parses Java source to find explicit implements declarations.
// Returns one ImplementsRef per interface per class.
func ExtractJavaImplementsRefs(src []byte) []kgdb.ImplementsRef {
return ExtractJava(src, 0, 0).ImplRefs
}

// ExtractJavaTypeRefs extracts non-call type references from Java source.
func ExtractJavaTypeRefs(src []byte) []kgdb.TypeRef {
return ExtractJava(src, 0, 0).TypeRefs
}
