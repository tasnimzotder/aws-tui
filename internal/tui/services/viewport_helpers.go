package services

import (
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

// NewStyledViewport creates a viewport with standard styling, mouse wheel, and
// soft wrap enabled. Width is clamped to a minimum of 80 and height to 1.
func NewStyledViewport(width, height int) viewport.Model {
	if width < 80 {
		width = 80
	}
	if height < 1 {
		height = 1
	}
	vp := viewport.New(
		viewport.WithWidth(width),
		viewport.WithHeight(height),
	)
	vp.MouseWheelEnabled = true
	vp.SoftWrap = true
	vp.Style = lipgloss.NewStyle().Padding(0, 1)
	return vp
}
