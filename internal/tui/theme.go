package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// Palette. Adaptive colours keep the editor readable on light terminals too,
// which a hard-coded dark theme does not.
var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#1b6ac9", Dark: "#7aa2f7"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#8b93a7"}
	colorText   = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#e5e9f0"}
	colorOK     = lipgloss.AdaptiveColor{Light: "#0f7b3d", Dark: "#9ece6a"}
	colorWarn   = lipgloss.AdaptiveColor{Light: "#a45c00", Dark: "#e0af68"}
	colorErr    = lipgloss.AdaptiveColor{Light: "#b3261e", Dark: "#f7768e"}
	colorPanel  = lipgloss.AdaptiveColor{Light: "#dfe3ea", Dark: "#2a2f3d"}
)

var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(colorAccent).
			Padding(0, 1)

	styleSubHeader = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleSidebar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(colorPanel).
			Padding(0, 1)

	styleNavItem = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	styleNavActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(colorAccent).
			Padding(0, 1)

	styleSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleStatusOK  = lipgloss.NewStyle().Foreground(colorOK).Padding(0, 1)
	styleStatusErr = lipgloss.NewStyle().Foreground(colorErr).Padding(0, 1)

	styleDirty = lipgloss.NewStyle().Foreground(colorWarn).Bold(true)

	styleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	styleModalTitle = lipgloss.NewStyle().Bold(true).Foreground(colorText)
	styleHint       = lipgloss.NewStyle().Foreground(colorMuted)
	styleWarn       = lipgloss.NewStyle().Foreground(colorWarn)
	styleMuted      = lipgloss.NewStyle().Foreground(colorMuted)
)

// tableStyles keeps every grid in the editor looking like the same widget.
func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorPanel).
		BorderBottom(true).
		Bold(true).
		Foreground(colorMuted)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffffff")).
		Background(colorAccent).
		Bold(true)
	s.Cell = s.Cell.Foreground(colorText)
	return s
}

func newTable(cols []table.Column, height int) table.Model {
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	t.SetStyles(tableStyles())
	return t
}
