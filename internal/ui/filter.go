package ui

import (
	"sort"
	"strings"
)

type scoredItem struct {
	value string
	score int
}

// FuzzyFilter filters and ranks items against a query string.
// Empty query returns all items unchanged. No matches returns nil.
// Scoring: exact match (100), prefix (75), word boundary (50), substring (25).
func FuzzyFilter(items []string, query string) []string {
	if query == "" {
		return items
	}

	lower := strings.ToLower(query)
	var scored []scoredItem

	for _, item := range items {
		itemLower := strings.ToLower(item)
		score := matchScore(itemLower, lower)
		if score > 0 {
			scored = append(scored, scoredItem{value: item, score: score})
		}
	}

	if len(scored) == 0 {
		return nil
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].value < scored[j].value
	})

	result := make([]string, len(scored))
	for i, s := range scored {
		result[i] = s.value
	}
	return result
}

func matchScore(item, query string) int {
	if item == query {
		return 100
	}

	if strings.HasPrefix(item, query) {
		return 75
	}

	// Word boundary: query appears after a separator character
	for _, sep := range []byte{'-', '_', '/', '.'} {
		idx := strings.Index(item, string(sep)+query)
		if idx >= 0 {
			return 50
		}
	}

	if strings.Contains(item, query) {
		return 25
	}

	return 0
}
