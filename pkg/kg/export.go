package kg

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	inquery "aikits/internal/kg/query"
)

// Export writes the knowledge graph to the destination specified in opts.
// If opts.Output is empty the graph is written to <repoRoot>/.kg/kg.<format>.
func (kg *KG) Export(_ context.Context, opts ExportOptions) error {
	format := opts.Format
	if format == "" {
		format = FormatJSON
	}

	output := opts.Output
	if output == "" {
		output = filepath.Join(kg.root, ".kg", fmt.Sprintf("kg.%s", string(format)))
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	return kg.ExportTo(context.Background(), f, opts)
}

// ExportTo writes the knowledge graph to w in the format specified by opts.Format.
func (kg *KG) ExportTo(_ context.Context, w io.Writer, opts ExportOptions) error {
	format := opts.Format
	if format == "" {
		format = FormatJSON
	}

	lang := string(opts.Lang)
	switch format {
	case FormatJSON:
		return exportJSON(w, kg.db, kg.repo.ID, kg.repo.Name, lang)
	case FormatGraphML:
		return exportGraphML(w, kg.db, kg.repo.ID, kg.repo.Name, lang)
	default:
		return fmt.Errorf("unsupported format %q; use %q or %q", format, FormatJSON, FormatGraphML)
	}
}

type jsonNode struct {
	ID         int64  `json:"id"`
	Lang       string `json:"lang"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	FQN        string `json:"fqn,omitempty"`
	Signature  string `json:"signature,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	StartLine  int    `json:"start_line"`
}

type jsonEdge struct {
	ID         int64   `json:"id"`
	Kind       string  `json:"kind"`
	Src        int64   `json:"src"`
	Dst        int64   `json:"dst"`
	Confidence float64 `json:"confidence"`
	Provenance string  `json:"provenance"`
}

func exportJSON(w io.Writer, db *sql.DB, repoID int64, repoName, lang string) error {
	enc := json.NewEncoder(w)

	repoJSON, _ := json.Marshal(repoName)
	if _, err := fmt.Fprintf(w, "{\"repo\":%s,\"nodes\":[", repoJSON); err != nil {
		return err
	}

	first := true
	if err := inquery.IterateSymbols(db, repoID, lang, func(s inquery.Symbol) error {
		if !first {
			if _, wErr := fmt.Fprint(w, ","); wErr != nil {
				return wErr
			}
		}
		first = false
		return enc.Encode(jsonNode{
			ID: s.ID, Lang: s.Lang, Kind: s.Kind, Name: s.Name,
			FQN: s.FQN, Signature: s.Signature, Visibility: s.Visibility,
			StartLine: s.StartLine,
		})
	}); err != nil {
		return fmt.Errorf("stream nodes: %w", err)
	}

	if _, err := fmt.Fprint(w, "],\"edges\":["); err != nil {
		return err
	}

	first = true
	if err := inquery.IterateEdges(db, repoID, func(e inquery.EdgeResult) error {
		if !first {
			if _, wErr := fmt.Fprint(w, ","); wErr != nil {
				return wErr
			}
		}
		first = false
		return enc.Encode(jsonEdge{
			ID: e.ID, Kind: e.Kind, Src: e.SrcSymbolID, Dst: e.DstSymbolID,
			Confidence: e.Confidence, Provenance: e.Provenance,
		})
	}); err != nil {
		return fmt.Errorf("stream edges: %w", err)
	}

	_, err := fmt.Fprintln(w, "]}")
	return err
}

type xmlKey struct {
	ID   string `xml:"id,attr"`
	For  string `xml:"for,attr"`
	Name string `xml:"attr.name,attr"`
	Type string `xml:"attr.type,attr"`
}

type xmlGMLNode struct {
	ID   string    `xml:"id,attr"`
	Data []xmlData `xml:"data"`
}

type xmlGMLEdge struct {
	ID     string    `xml:"id,attr"`
	Source string    `xml:"source,attr"`
	Target string    `xml:"target,attr"`
	Data   []xmlData `xml:"data"`
}

type xmlData struct {
	Key   string `xml:"key,attr"`
	Value string `xml:",chardata"`
}

func exportGraphML(w io.Writer, db *sql.DB, repoID int64, repoName, lang string) error {
	if _, err := fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")

	start := xml.StartElement{
		Name: xml.Name{Local: "graphml"},
		Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: "http://graphml.graphdrawing.org/graphml"}},
	}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}

	for _, k := range []xmlKey{
		{ID: "node_kind", For: "node", Name: "kind", Type: "string"},
		{ID: "name", For: "node", Name: "name", Type: "string"},
		{ID: "fqn", For: "node", Name: "fqn", Type: "string"},
		{ID: "lang", For: "node", Name: "lang", Type: "string"},
		{ID: "edge_kind", For: "edge", Name: "kind", Type: "string"},
	} {
		if err := enc.EncodeElement(k, xml.StartElement{Name: xml.Name{Local: "key"}}); err != nil {
			return err
		}
	}

	graphStart := xml.StartElement{
		Name: xml.Name{Local: "graph"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: repoName},
			{Name: xml.Name{Local: "edgedefault"}, Value: "directed"},
		},
	}
	if err := enc.EncodeToken(graphStart); err != nil {
		return err
	}

	if err := inquery.IterateSymbols(db, repoID, lang, func(s inquery.Symbol) error {
		return enc.EncodeElement(xmlGMLNode{
			ID: fmt.Sprintf("n%d", s.ID),
			Data: []xmlData{
				{Key: "node_kind", Value: s.Kind},
				{Key: "name", Value: s.Name},
				{Key: "fqn", Value: s.FQN},
				{Key: "lang", Value: s.Lang},
			},
		}, xml.StartElement{Name: xml.Name{Local: "node"}})
	}); err != nil {
		return fmt.Errorf("stream nodes: %w", err)
	}

	if err := inquery.IterateEdges(db, repoID, func(e inquery.EdgeResult) error {
		return enc.EncodeElement(xmlGMLEdge{
			ID:     fmt.Sprintf("e%d", e.ID),
			Source: fmt.Sprintf("n%d", e.SrcSymbolID),
			Target: fmt.Sprintf("n%d", e.DstSymbolID),
			Data:   []xmlData{{Key: "edge_kind", Value: e.Kind}},
		}, xml.StartElement{Name: xml.Name{Local: "edge"}})
	}); err != nil {
		return fmt.Errorf("stream edges: %w", err)
	}

	if err := enc.EncodeToken(graphStart.End()); err != nil {
		return err
	}
	if err := enc.EncodeToken(start.End()); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}
