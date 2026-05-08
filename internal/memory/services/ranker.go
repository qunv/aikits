package services

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"aikits/internal/memory/types"
)

// RankContext carries optional contextual hints that boost result scores.
type RankContext struct {
	ContextTags []string
	QueryScope  string
}

// RankResults applies the scoring formula and returns ranked SearchResultItems.
//
// Formula: finalScore = (-bm25Score × tagBoost) + scopeBoost
// BM25 returns negative values (closer to 0 = better), so we negate it.
func RankResults(rows []types.RawSearchRow, ctx RankContext) []types.SearchResultItem {
	items := make([]types.SearchResultItem, 0, len(rows))

	for _, row := range rows {
		var tags []string
		if err := json.Unmarshal([]byte(row.Tags), &tags); err != nil {
			tags = []string{}
		}

		tagBoost := calcTagBoost(tags, ctx.ContextTags)
		scopeBoost := calcScopeBoost(row.Scope, ctx.QueryScope)

		normalised := -row.BM25Score // negate: higher = better
		score := (normalised * tagBoost) + scopeBoost
		score = math.Round(score*1000) / 1000

		items = append(items, types.SearchResultItem{
			ID:      row.ID,
			Title:   row.Title,
			Content: row.Content,
			Tags:    tags,
			Scope:   row.Scope,
			Score:   score,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	return items
}

// calcTagBoost returns 1.0 + 0.1 per matching context tag.
func calcTagBoost(itemTags, contextTags []string) float64 {
	if len(contextTags) == 0 {
		return 1.0
	}
	lower := make([]string, len(itemTags))
	for i, t := range itemTags {
		lower[i] = strings.ToLower(t)
	}
	matches := 0
	for _, ct := range contextTags {
		ct = strings.ToLower(ct)
		for _, it := range lower {
			if it == ct {
				matches++
				break
			}
		}
	}
	return 1.0 + float64(matches)*0.1
}

// calcScopeBoost returns +0.5 for an exact scope match, +0.2 for global.
func calcScopeBoost(itemScope, queryScope string) float64 {
	if queryScope != "" && itemScope == queryScope {
		return 0.5
	}
	if itemScope == "global" {
		return 0.2
	}
	return 0
}
