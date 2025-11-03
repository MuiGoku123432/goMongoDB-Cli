package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MenuModel struct {
	choices  []string
	cursor   int
	selected int
	width    int
	height   int
}

func NewMenuModel() *MenuModel {
	return &MenuModel{
		choices: []string{
			"📥 Import CSV to MongoDB",
			"💾 Backup MongoDB Collection",
			"🔄 Restore MongoDB Collection", 
			"⚙️  Settings",
			"🚪 Exit",
		},
		cursor: 0,
	}
}

func (m *MenuModel) Init() tea.Cmd {
	return nil
}

func (m *MenuModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			return m, m.handleSelection()
		}
	}
	return m, nil
}

func (m *MenuModel) handleSelection() tea.Cmd {
	switch m.selected {
	case 0: // Import
		return ChangeScreen(ImportScreen)
	case 1: // Backup
		return ChangeScreen(BackupScreen)
	case 2: // Restore
		return ChangeScreen(RestoreScreen)
	case 3: // Settings
		return ChangeScreen(SettingsScreen)
	case 4: // Exit
		return tea.Quit
	}
	return nil
}

func (m *MenuModel) View() string {
	adaptiveTitleStyle, _, adaptiveHelpStyle := GetAdaptiveStyles(m.width, m.height)
	
	title := adaptiveTitleStyle.Render("🗄️  MongoDB TUI - CSV Processor")
	
	var menu string
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			choice = selectedMenuItemStyle.Render(choice)
		} else {
			choice = menuItemStyle.Render(choice)
		}
		menu += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	help := adaptiveHelpStyle.Render("Use ↑/↓ (or j/k) to navigate • Enter to select • q to quit")
	
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		menu,
		help,
	)
	
	// Center the content in the available space
	if m.width > 0 {
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			content,
		)
	}
	
	return content
}