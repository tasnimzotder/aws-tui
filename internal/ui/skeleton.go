package ui

import "strings"

// Skeleton renders placeholder loading lines using block characters.
type Skeleton struct {
	width int
	lines int
}

// NewSkeleton creates a Skeleton with the given width and number of lines.
func NewSkeleton(width, lines int) Skeleton {
	return Skeleton{width: width, lines: lines}
}

// View returns the skeleton placeholder text.
// Even-indexed lines are full width; odd-indexed lines are 2/3 width.
func (s Skeleton) View() string {
	var b strings.Builder
	for i := 0; i < s.lines; i++ {
		w := s.width
		if i%2 == 1 {
			w = s.width * 2 / 3
		}
		b.WriteString(strings.Repeat("░", w))
		if i < s.lines-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
