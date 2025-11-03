package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen int

const (
	MenuScreen Screen = iota
	ImportScreen
	BackupScreen
	RestoreScreen
	SettingsScreen
)

type Model struct {
	currentScreen Screen
	menuModel     *MenuModel
	importModel   *ImportModel
	backupModel   *BackupModel
	restoreModel  *RestoreModel
	err           error
	quitting      bool
	width         int
	height        int
}

func NewModel() Model {
	return Model{
		currentScreen: MenuScreen,
		menuModel:     NewMenuModel(),
		importModel:   NewImportModel(),
		backupModel:   NewBackupModel(),
		restoreModel:  NewRestoreModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Pass window size to sub-models
		m.menuModel.SetSize(msg.Width, msg.Height)
		m.importModel.SetSize(msg.Width, msg.Height)
		m.backupModel.SetSize(msg.Width, msg.Height)
		m.restoreModel.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			if m.currentScreen != MenuScreen {
				m.currentScreen = MenuScreen
				return m, nil
			}
		}

	case ScreenChangeMsg:
		m.currentScreen = msg.Screen
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	switch m.currentScreen {
	case MenuScreen:
		newMenuModel, cmd := m.menuModel.Update(msg)
		m.menuModel = newMenuModel.(*MenuModel)
		return m, cmd
	case ImportScreen:
		newImportModel, cmd := m.importModel.Update(msg)
		m.importModel = newImportModel.(*ImportModel)
		return m, cmd
	case BackupScreen:
		newBackupModel, cmd := m.backupModel.Update(msg)
		m.backupModel = newBackupModel.(*BackupModel)
		return m, cmd
	case RestoreScreen:
		newRestoreModel, cmd := m.restoreModel.Update(msg)
		m.restoreModel = newRestoreModel.(*RestoreModel)
		return m, cmd
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return "Thanks for using MongoDB TUI! 👋\n"
	}

	var content string
	switch m.currentScreen {
	case MenuScreen:
		content = m.menuModel.View()
	case ImportScreen:
		content = m.importModel.View()
	case BackupScreen:
		content = m.backupModel.View()
	case RestoreScreen:
		content = m.restoreModel.View()
	}

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Margin(1, 0)
		content += errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	return content
}

type ScreenChangeMsg struct {
	Screen Screen
}

type ErrorMsg struct {
	Err error
}

func ChangeScreen(screen Screen) tea.Cmd {
	return func() tea.Msg {
		return ScreenChangeMsg{Screen: screen}
	}
}

func ShowError(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Err: err}
	}
}