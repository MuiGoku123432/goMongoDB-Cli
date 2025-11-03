package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"excelDisclaimer/internal/database"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type RestoreModel struct {
	state           RestoreState
	backupFileInput textinput.Model
	dbURIInput      textinput.Model
	dbNameInput     textinput.Model
	collectionInput textinput.Model
	focusedInput    int
	formatSelection int
	formats         []string
	dropExisting    bool
	progress        progress.Model
	progressVal     float64
	restoring       bool
	completed       bool
	result          RestoreResult
	files           []string
	selectedFile    int
	showingFiles    bool
	showingConfirm  bool
	width           int
	height          int
}

type RestoreState int

const (
	RestoreInputState RestoreState = iota
	RestoreFileSelectState
	RestoreFormatSelectState
	ConfirmationState
	RestoreProgressState
	RestoreResultState
)

type RestoreResult struct {
	DocumentCount int
	Error         error
}

type RestoreProgressMsg struct {
	Progress float64
	Status   string
}

type RestoreCompleteMsg struct {
	Result RestoreResult
}

func NewRestoreModel() *RestoreModel {
	backupFileInput := textinput.New()
	backupFileInput.Placeholder = "backup.json"
	backupFileInput.Focus()

	dbURIInput := textinput.New()
	dbURIInput.Placeholder = "mongodb://localhost:27017"
	dbURIInput.SetValue("mongodb://localhost:27017")

	dbNameInput := textinput.New()
	dbNameInput.Placeholder = "csvprocessor"
	dbNameInput.SetValue("csvprocessor")

	collectionInput := textinput.New()
	collectionInput.Placeholder = "records"
	collectionInput.SetValue("records")

	progressBar := progress.New(
		progress.WithSolidFill("#00aadd"),
		progress.WithoutPercentage(),
	)
	
	return &RestoreModel{
		state:           RestoreInputState,
		backupFileInput: backupFileInput,
		dbURIInput:      dbURIInput,
		dbNameInput:     dbNameInput,
		collectionInput: collectionInput,
		focusedInput:    0,
		formatSelection: 0,
		formats:         []string{"JSON", "BSON"},
		dropExisting:    false,
		progress:        progressBar,
	}
}

func (m *RestoreModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RestoreModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *RestoreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case RestoreInputState:
			return m.updateInputState(msg)
		case RestoreFileSelectState:
			return m.updateFileSelectState(msg)
		case RestoreFormatSelectState:
			return m.updateFormatSelectState(msg)
		case ConfirmationState:
			return m.updateConfirmationState(msg)
		case RestoreProgressState:
			// No input during progress
			return m, nil
		case RestoreResultState:
			if msg.String() == "enter" || msg.String() == " " {
				m.reset()
				return m, nil
			}
		}

	case RestoreProgressMsg:
		m.progressVal = msg.Progress
		return m, nil

	case RestoreCompleteMsg:
		m.result = msg.Result
		m.restoring = false
		m.completed = true
		m.state = RestoreResultState
		return m, nil
	}

	return m, cmd
}

func (m *RestoreModel) updateInputState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "tab", "down":
		m.focusedInput = (m.focusedInput + 1) % 4
		m.updateInputFocus()
	case "shift+tab", "up":
		m.focusedInput = (m.focusedInput - 1 + 4) % 4
		m.updateInputFocus()
	case "ctrl+f":
		return m.browseFiles()
	case "enter":
		if m.isFormValid() {
			m.state = RestoreFormatSelectState
			return m, nil
		}
	}

	switch m.focusedInput {
	case 0:
		m.backupFileInput, cmd = m.backupFileInput.Update(msg)
	case 1:
		m.dbURIInput, cmd = m.dbURIInput.Update(msg)
	case 2:
		m.dbNameInput, cmd = m.dbNameInput.Update(msg)
	case 3:
		m.collectionInput, cmd = m.collectionInput.Update(msg)
	}

	return m, cmd
}

func (m *RestoreModel) updateFileSelectState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedFile > 0 {
			m.selectedFile--
		}
	case "down", "j":
		if m.selectedFile < len(m.files)-1 {
			m.selectedFile++
		}
	case "enter":
		if len(m.files) > 0 {
			m.backupFileInput.SetValue(m.files[m.selectedFile])
			m.state = RestoreInputState
			m.showingFiles = false
		}
	case "esc":
		m.state = RestoreInputState
		m.showingFiles = false
	}
	return m, nil
}

func (m *RestoreModel) updateFormatSelectState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.formatSelection > 0 {
			m.formatSelection--
		}
	case "down", "j":
		if m.formatSelection < len(m.formats)-1 {
			m.formatSelection++
		}
	case "enter":
		m.state = ConfirmationState
		m.showingConfirm = true
		return m, nil
	case "esc":
		m.state = RestoreInputState
	}
	return m, nil
}

func (m *RestoreModel) updateConfirmationState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "d":
		m.dropExisting = !m.dropExisting
	case "y", "enter":
		return m.startRestore()
	case "n", "esc":
		m.state = RestoreInputState
		m.showingConfirm = false
	}
	return m, nil
}

func (m *RestoreModel) browseFiles() (tea.Model, tea.Cmd) {
	cwd, _ := os.Getwd()
	jsonFiles, _ := filepath.Glob(filepath.Join(cwd, "*.json"))
	bsonFiles, _ := filepath.Glob(filepath.Join(cwd, "*.bson"))
	
	files := append(jsonFiles, bsonFiles...)

	// Convert to relative paths
	for i, file := range files {
		rel, _ := filepath.Rel(cwd, file)
		files[i] = rel
	}

	m.files = files
	m.selectedFile = 0
	m.state = RestoreFileSelectState
	m.showingFiles = true
	return m, nil
}

func (m *RestoreModel) updateInputFocus() {
	inputs := []*textinput.Model{&m.backupFileInput, &m.dbURIInput, &m.dbNameInput, &m.collectionInput}
	for i, input := range inputs {
		if i == m.focusedInput {
			input.Focus()
		} else {
			input.Blur()
		}
	}
}

func (m *RestoreModel) isFormValid() bool {
	return strings.TrimSpace(m.backupFileInput.Value()) != "" &&
		strings.TrimSpace(m.dbURIInput.Value()) != "" &&
		strings.TrimSpace(m.dbNameInput.Value()) != "" &&
		strings.TrimSpace(m.collectionInput.Value()) != ""
}

func (m *RestoreModel) startRestore() (tea.Model, tea.Cmd) {
	m.state = RestoreProgressState
	m.restoring = true
	m.progressVal = 0
	m.showingConfirm = false
	return m, m.performRestore()
}

func (m *RestoreModel) performRestore() tea.Cmd {
	return func() tea.Msg {
		backupFile := strings.TrimSpace(m.backupFileInput.Value())
		dbURI := strings.TrimSpace(m.dbURIInput.Value())
		dbName := strings.TrimSpace(m.dbNameInput.Value())
		collection := strings.TrimSpace(m.collectionInput.Value())
		format := strings.ToLower(m.formats[m.formatSelection])

		result := RestoreResult{}

		// Connect to database
		db, err := database.NewMongoDB(dbURI, dbName)
		if err != nil {
			result.Error = fmt.Errorf("failed to connect to MongoDB: %w", err)
			return RestoreCompleteMsg{Result: result}
		}
		defer db.Close()

		// Open backup file
		file, err := os.Open(backupFile)
		if err != nil {
			result.Error = fmt.Errorf("failed to open backup file: %w", err)
			return RestoreCompleteMsg{Result: result}
		}
		defer file.Close()

		// Perform restore
		err = db.RestoreCollection(collection, file, format, m.dropExisting)
		if err != nil {
			result.Error = fmt.Errorf("restore failed: %w", err)
			return RestoreCompleteMsg{Result: result}
		}

		// Get rough document count (simplified)
		fileInfo, _ := file.Stat()
		result.DocumentCount = int(fileInfo.Size() / 100) // Rough estimate

		return RestoreCompleteMsg{Result: result}
	}
}

func (m *RestoreModel) reset() {
	m.state = RestoreInputState
	m.restoring = false
	m.completed = false
	m.progressVal = 0
	m.result = RestoreResult{}
	m.dropExisting = false
	m.showingConfirm = false
	m.updateInputFocus()
}

func (m *RestoreModel) View() string {
	switch m.state {
	case RestoreInputState:
		return m.renderInputForm()
	case RestoreFileSelectState:
		return m.renderFileSelector()
	case RestoreFormatSelectState:
		return m.renderFormatSelector()
	case ConfirmationState:
		return m.renderConfirmation()
	case RestoreProgressState:
		return m.renderProgress()
	case RestoreResultState:
		return m.renderResult()
	}
	return ""
}

func (m *RestoreModel) renderInputForm() string {
	title := titleStyle.Render("🔄 Restore MongoDB Collection")

	form := formStyle.Render(
		labelStyle.Render("Backup File:") + "\n" + m.backupFileInput.View() + "\n\n" +
		labelStyle.Render("Database URI:") + "\n" + m.dbURIInput.View() + "\n\n" +
		labelStyle.Render("Database Name:") + "\n" + m.dbNameInput.View() + "\n\n" +
		labelStyle.Render("Collection:") + "\n" + m.collectionInput.View(),
	)

	help := helpStyle.Render("Tab/Shift+Tab: Navigate • Ctrl+F: Browse files • Enter: Continue • Esc: Back to menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, form, help)
}

func (m *RestoreModel) renderFileSelector() string {
	title := titleStyle.Render("📁 Select Backup File")

	if len(m.files) == 0 {
		content := warningStyle.Render("No backup files (*.json, *.bson) found in current directory")
		help := helpStyle.Render("Esc: Back to form")
		return lipgloss.JoinVertical(lipgloss.Left, title, content, help)
	}

	var fileList string
	for i, file := range m.files {
		cursor := " "
		style := menuItemStyle
		if i == m.selectedFile {
			cursor = ">"
			style = selectedMenuItemStyle
		}
		fileList += fmt.Sprintf("%s %s\n", cursor, style.Render(file))
	}

	help := helpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Cancel")

	return lipgloss.JoinVertical(lipgloss.Left, title, fileList, help)
}

func (m *RestoreModel) renderFormatSelector() string {
	title := titleStyle.Render("📄 Select Backup Format")

	var formatList string
	for i, format := range m.formats {
		cursor := " "
		style := menuItemStyle
		if i == m.formatSelection {
			cursor = ">"
			style = selectedMenuItemStyle
		}
		formatList += fmt.Sprintf("%s %s\n", cursor, style.Render(format))
	}

	help := helpStyle.Render("↑/↓: Navigate • Enter: Continue • Esc: Back")

	return lipgloss.JoinVertical(lipgloss.Left, title, formatList, help)
}

func (m *RestoreModel) renderConfirmation() string {
	title := titleStyle.Render("⚠️  Confirm Restore Operation")

	warningText := warningStyle.Render("This operation will restore data to the specified collection.")
	
	dropText := "Drop existing collection: "
	if m.dropExisting {
		dropText += successStyle.Render("✓ YES")
	} else {
		dropText += errorStyle.Render("✗ NO")
	}

	details := fmt.Sprintf(
		"📋 Restore Details:\n"+
		"   File: %s\n"+
		"   Format: %s\n"+
		"   Database: %s\n"+
		"   Collection: %s\n"+
		"   %s",
		m.backupFileInput.Value(),
		m.formats[m.formatSelection],
		m.dbNameInput.Value(),
		m.collectionInput.Value(),
		dropText,
	)

	help := helpStyle.Render("D: Toggle drop existing • Y/Enter: Confirm • N/Esc: Cancel")

	return lipgloss.JoinVertical(lipgloss.Left, title, warningText, details, help)
}

func (m *RestoreModel) renderProgress() string {
	title := titleStyle.Render("🔄 Restoring Data...")
	
	progressBar := m.progress.ViewAs(m.progressVal)
	progressText := fmt.Sprintf("Progress: %.1f%%", m.progressVal*100)
	
	content := progressStyle.Render(progressBar + "\n" + progressText)
	help := helpStyle.Render("Please wait while data is being restored...")

	return lipgloss.JoinVertical(lipgloss.Left, title, content, help)
}

func (m *RestoreModel) renderResult() string {
	title := titleStyle.Render("🔄 Restore Complete")

	var status string
	if m.result.Error != nil {
		status = errorStyle.Render(fmt.Sprintf("❌ Restore failed: %v", m.result.Error))
	} else {
		status = successStyle.Render("✅ Restore completed successfully!")
	}

	stats := fmt.Sprintf(
		"📊 Restore Information:\n"+
		"   Source file: %s\n"+
		"   Format: %s\n"+
		"   Estimated documents: %d",
		m.backupFileInput.Value(),
		m.formats[m.formatSelection],
		m.result.DocumentCount,
	)

	help := helpStyle.Render("Enter: Restore another file • Esc: Back to menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, status, stats, help)
}