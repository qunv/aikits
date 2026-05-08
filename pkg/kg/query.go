package kg

import (
	"context"
	"strings"

	inquery "aikits/internal/kg/query"
)

// QuerySymbol looks up symbols by name or fully-qualified name.
// If nameOrFQN contains a dot it is treated as an FQN first; a name search
// is performed as a fallback when the FQN lookup returns no results.
func (kg *KG) QuerySymbol(_ context.Context, nameOrFQN string) ([]Symbol, error) {
	var syms []inquery.Symbol
	var err error

	if strings.Contains(nameOrFQN, ".") {
		syms, err = inquery.LookupByFQN(kg.db, kg.repo.ID, nameOrFQN)
		if err != nil {
			return nil, err
		}
	}
	if len(syms) == 0 {
		syms, err = inquery.LookupByName(kg.db, kg.repo.ID, nameOrFQN)
		if err != nil {
			return nil, err
		}
	}
	return mapSymbols(syms), nil
}

// Callers returns the symbols that call the given FQN (inbound CALLS edges).
func (kg *KG) Callers(_ context.Context, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	nodes, err := inquery.Callers(kg.db, kg.repo.ID, fqn, depth, maxNodes)
	if err != nil {
		return nil, err
	}
	return mapNodes(nodes), nil
}

// Callees returns the symbols that the given FQN calls (outbound CALLS edges).
func (kg *KG) Callees(_ context.Context, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	nodes, err := inquery.Callees(kg.db, kg.repo.ID, fqn, depth, maxNodes)
	if err != nil {
		return nil, err
	}
	return mapNodes(nodes), nil
}

// Impact returns symbols that depend on the given FQN (inbound CALLS +
// REFERENCES edges).
func (kg *KG) Impact(_ context.Context, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	nodes, err := inquery.Impact(kg.db, kg.repo.ID, fqn, depth, maxNodes)
	if err != nil {
		return nil, err
	}
	return mapNodes(nodes), nil
}

// Impls returns implementations of the given interface FQN (inbound IMPLEMENTS
// edges).
func (kg *KG) Impls(_ context.Context, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	nodes, err := inquery.Impls(kg.db, kg.repo.ID, fqn, depth, maxNodes)
	if err != nil {
		return nil, err
	}
	return mapNodes(nodes), nil
}

// Overrides returns symbols that override the given method FQN (inbound
// OVERRIDES edges).
func (kg *KG) Overrides(_ context.Context, fqn string, depth, maxNodes int) ([]SymbolNode, error) {
	nodes, err := inquery.Overrides(kg.db, kg.repo.ID, fqn, depth, maxNodes)
	if err != nil {
		return nil, err
	}
	return mapNodes(nodes), nil
}

func mapSymbol(s inquery.Symbol) Symbol {
	return Symbol{
		ID:         s.ID,
		Lang:       s.Lang,
		Kind:       s.Kind,
		Name:       s.Name,
		FQN:        s.FQN,
		Signature:  s.Signature,
		Visibility: s.Visibility,
		StartLine:  s.StartLine,
	}
}

func mapSymbols(ss []inquery.Symbol) []Symbol {
	out := make([]Symbol, len(ss))
	for i, s := range ss {
		out[i] = mapSymbol(s)
	}
	return out
}

func mapNodes(ns []inquery.SymbolNode) []SymbolNode {
	out := make([]SymbolNode, len(ns))
	for i, n := range ns {
		out[i] = SymbolNode{Symbol: mapSymbol(n.Symbol), Depth: n.Depth}
	}
	return out
}
