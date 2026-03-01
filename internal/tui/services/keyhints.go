package services

import (
	"charm.land/lipgloss/v2"

	"tasnim.dev/aws-tui/internal/tui/theme"
)

type keyHint struct {
	key  string
	desc string
}

// RenderKeyHints renders a compact one-line footer with key hints appropriate
// for the given help context. Hints are truncated to fit the given width.
func RenderKeyHints(ctx HelpContext, width int) string {
	hints := hintsForContext(ctx)

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	descStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	sepStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	sep := sepStyle.Render(" · ")

	var result string
	for i, h := range hints {
		part := keyStyle.Render(h.key) + " " + descStyle.Render(h.desc)
		candidate := result
		if i > 0 {
			candidate += sep
		}
		candidate += part
		// Rough length check (ANSI codes inflate actual length)
		plainLen := len(h.key) + 1 + len(h.desc)
		if i > 0 {
			plainLen += 3 // " · "
		}
		totalPlain := estimatePlainLen(result) + plainLen
		if totalPlain > width && i > 0 {
			break
		}
		if i > 0 {
			result += sep
		}
		result += part
	}
	return result
}

func estimatePlainLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		n++
	}
	return n
}

func hintsForContext(ctx HelpContext) []keyHint {
	switch ctx {
	case HelpContextRoot:
		return []keyHint{
			{"Enter", "select"},
			{"j/k", "navigate"},
			{"?", "help"},
			{"q", "quit"},
		}
	case HelpContextTable:
		return []keyHint{
			{"Enter", "drill"},
			{"/", "filter"},
			{"n/p", "page"},
			{"r", "refresh"},
			{"a", "auto-refresh"},
			{"c", "copy"},
			{"?", "help"},
		}
	case HelpContextDetail:
		return []keyHint{
			{"Tab", "switch"},
			{"r", "refresh"},
			{"c", "copy"},
			{"Esc", "back"},
			{"?", "help"},
		}
	case HelpContextEC2Detail:
		return []keyHint{
			{"Tab", "switch"},
			{"x", "SSM"},
			{"v", "VPC"},
			{"Esc", "back"},
			{"?", "help"},
		}
	case HelpContextELBDetail:
		return []keyHint{
			{"Tab", "switch"},
			{"v", "VPC"},
			{"c", "copy"},
			{"Esc", "back"},
			{"?", "help"},
		}
	case HelpContextVPCDetail:
		return []keyHint{
			{"Tab", "switch"},
			{"1-9/0", "jump"},
			{"c", "copy"},
			{"Esc", "back"},
			{"?", "help"},
		}
	case HelpContextEKSDetail:
		return []keyHint{
			{"Tab", "switch"},
			{"N", "namespace"},
			{"c", "copy"},
			{"Esc", "back"},
			{"?", "help"},
		}
	case HelpContextK8sPods:
		return []keyHint{
			{"Enter", "details"},
			{"l", "logs"},
			{"x", "exec"},
			{"f", "port-fwd"},
			{"/", "filter"},
			{"?", "help"},
		}
	case HelpContextK8sNodes:
		return []keyHint{
			{"Enter", "details"},
			{"e", "YAML"},
			{"x", "debug"},
			{"/", "filter"},
			{"?", "help"},
		}
	case HelpContextK8sLogs:
		return []keyHint{
			{"f", "follow"},
			{"w", "wrap"},
			{"/", "search"},
			{"n/N", "next/prev"},
			{"Esc", "back"},
		}
	case HelpContextTextView:
		return []keyHint{
			{"/", "search"},
			{"n/N", "next/prev"},
			{"w", "wrap"},
			{"Esc", "back"},
			{"?", "help"},
		}
	case HelpContextS3Objects:
		return []keyHint{
			{"Enter", "open"},
			{"v", "view"},
			{"d", "download"},
			{"/", "filter"},
			{"?", "help"},
		}
	default:
		return []keyHint{
			{"Esc", "back"},
			{"?", "help"},
			{"q", "quit"},
		}
	}
}
