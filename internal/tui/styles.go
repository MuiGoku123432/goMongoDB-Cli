package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
		Bold(true).
		Margin(1, 0, 2, 0).
		Align(lipgloss.Center)

	menuItemStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Margin(0, 1).
		Foreground(lipgloss.AdaptiveColor{Light: "#262626", Dark: "#d9d9d9"})

	selectedMenuItemStyle = menuItemStyle.Copy().
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#000000"}).
		Background(lipgloss.AdaptiveColor{Light: "#005577", Dark: "#00aadd"}).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#626262", Dark: "#a8a8a8"}).
		Margin(2, 0, 0, 0)

	formStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#005577", Dark: "#00aadd"}).
		Padding(1, 2).
		Margin(1, 0)

	inputStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#d33682", Dark: "#ff79c6"})

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#859900", Dark: "#50fa7b"}).
		Bold(true)

	progressStyle = lipgloss.NewStyle().
		Margin(1, 0)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#859900", Dark: "#50fa7b"}).
		Bold(true)

	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#b58900", Dark: "#f1fa8c"}).
		Bold(true)

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#dc322f", Dark: "#ff5555"}).
		Bold(true)
)

// GetAdaptiveStyles returns styles that adapt to terminal width
func GetAdaptiveStyles(width, height int) (titleStyle, formStyle, helpStyle lipgloss.Style) {
	maxWidth := width - 4 // Leave some margin
	
	adaptiveTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
		Bold(true).
		Margin(1, 0, 2, 0).
		Align(lipgloss.Center).
		Width(maxWidth)

	adaptiveFormStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#005577", Dark: "#00aadd"}).
		Padding(1, 2).
		Margin(1, 0).
		Width(maxWidth)

	adaptiveHelpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#626262", Dark: "#a8a8a8"}).
		Margin(2, 0, 0, 0).
		Width(maxWidth)

	return adaptiveTitleStyle, adaptiveFormStyle, adaptiveHelpStyle
}