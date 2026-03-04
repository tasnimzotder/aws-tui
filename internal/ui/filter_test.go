package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFuzzyFilter(t *testing.T) {
	items := []string{
		"us-east-1",
		"us-west-2",
		"eu-west-1",
		"ap-southeast-1",
		"eu-central-1",
	}

	tests := []struct {
		name     string
		items    []string
		query    string
		expected []string
	}{
		{
			name:     "empty query returns all items",
			items:    items,
			query:    "",
			expected: items,
		},
		{
			name:     "exact prefix match",
			items:    items,
			query:    "us-east-1",
			expected: []string{"us-east-1"},
		},
		{
			name:     "substring match",
			items:    items,
			query:    "west",
			expected: []string{"eu-west-1", "us-west-2"},
		},
		{
			name:     "case insensitive",
			items:    items,
			query:    "US-EAST",
			expected: []string{"us-east-1"},
		},
		{
			name:     "no match returns nil",
			items:    items,
			query:    "xyz",
			expected: nil,
		},
		{
			name:  "ranking: exact > prefix > word boundary > substring",
			items: []string{"east-config", "us-east-1", "east", "southeast"},
			query: "east",
			expected: []string{
				"east",           // exact match (100)
				"east-config",    // prefix match (75)
				"us-east-1",     // word boundary match after '-' (50)
				"southeast",      // substring match (25)
			},
		},
		{
			name:     "nil items returns nil for non-empty query",
			items:    nil,
			query:    "test",
			expected: nil,
		},
		{
			name:     "empty items with empty query",
			items:    []string{},
			query:    "",
			expected: []string{},
		},
		{
			name:  "word boundary match after various separators",
			items: []string{"my_east", "my.east", "my/east", "my-east"},
			query: "east",
			expected: []string{
				"my-east",
				"my.east",
				"my/east",
				"my_east",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FuzzyFilter(tt.items, tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
