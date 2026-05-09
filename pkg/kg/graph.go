package kg

import (
	"context"

	inquery "aikits/internal/kg/query"
)

// GetGraph returns all nodes and edges in the knowledge graph for the given repo.
// lang filters by language; empty string returns all languages.
func (kg *KG) GetGraph(_ context.Context, lang string) (*GraphData, error) {
	var nodes []GraphNode
	if err := inquery.IterateSymbols(kg.db, kg.repo.ID, lang, func(s inquery.Symbol) error {
		nodes = append(nodes, GraphNode{
			ID:         s.ID,
			Lang:       s.Lang,
			Kind:       s.Kind,
			Name:       s.Name,
			FQN:        s.FQN,
			Signature:  s.Signature,
			Visibility: s.Visibility,
			StartLine:  s.StartLine,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	var edges []GraphEdge
	if err := inquery.IterateEdges(kg.db, kg.repo.ID, func(e inquery.EdgeResult) error {
		edges = append(edges, GraphEdge{
			ID:         e.ID,
			Kind:       e.Kind,
			Src:        e.SrcSymbolID,
			Dst:        e.DstSymbolID,
			Confidence: e.Confidence,
			Provenance: e.Provenance,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	if nodes == nil {
		nodes = []GraphNode{}
	}
	if edges == nil {
		edges = []GraphEdge{}
	}

	return &GraphData{Nodes: nodes, Edges: edges}, nil
}
