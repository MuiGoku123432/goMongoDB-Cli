package tui

import (
	"fmt"
	"os"
	"strings"

	"excelDisclaimer/internal/database"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type BackupModel struct {
	state            BackupState
	dbURIInput       textinput.Model
	dbNameInput      textinput.Model
	collectionInput  textinput.Model
	outputFileInput  textinput.Model
	focusedInput     int
	formatSelection  int
	formats          []string
	progress         progress.Model
	progressVal      float64
	backing          bool
	completed        bool
	result           BackupResult
	collections      []string
	selectedColl     int
	showingColls     bool
	width            int
	height           int
}

type BackupState int

const (
	BackupInputState BackupState = iota
	BackupCollectionSelectState
	BackupFormatSelectState
	BackupProgressState
	BackupResultState
)

type BackupResult struct {
	DocumentCount int
	FilePath      string
	Error         error
}

type BackupProgressMsg struct {
	Progress float64
	Status   string
}

type BackupCompleteMsg struct {
	Result BackupResult
}

func NewBackupModel() *BackupModel {
	dbURIInput := textinput.New()
	dbURIInput.Placeholder = "mongodb://localhost:27017"
	dbURIInput.SetValue("mongodb://localhost:27017")
	dbURIInput.Focus()

	dbNameInput := textinput.New()
	dbNameInput.Placeholder = "csvprocessor"
	dbNameInput.SetValue("csvprocessor")

	collectionInput := textinput.New()
	collectionInput.Placeholder = "records"
	collectionInput.SetValue("records")

	outputFileInput := textinput.New()
	outputFileInput.Placeholder = "backup.json"
	outputFileInput.SetValue("backup.json")

	progressBar := progress.New(
		progress.WithSolidFill("#00aadd"),
		progress.WithoutPercentage(),
	)
	
	return &BackupModel{
		state:           BackupInputState,
		dbURIInput:      dbURIInput,
		dbNameInput:     dbNameInput,
		collectionInput: collectionInput,
		outputFileInput: outputFileInput,
		focusedInput:    0,
		formatSelection: 0,
		formats:         []string{"JSON", "BSON"},
		progress:        progressBar,
	}
}

func (m *BackupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *BackupModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case BackupInputState:
			return m.updateInputState(msg)
		case BackupCollectionSelectState:
			return m.updateCollectionSelectState(msg)
		case BackupFormatSelectState:
			return m.updateFormatSelectState(msg)
		case BackupProgressState:
			// No input during progress
			return m, nil
		case BackupResultState:
			if msg.String() == "enter" || msg.String() == " " {
				m.reset()
				return m, nil
			}
		}

	case BackupProgressMsg:
		m.progressVal = msg.Progress
		return m, nil

	case BackupCompleteMsg:
		m.result = msg.Result
		m.backing = false
		m.completed = true
		m.state = BackupResultState
		return m, nil
	}

	return m, cmd
}

func (m *BackupModel) updateInputState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "tab", "down":
		m.focusedInput = (m.focusedInput + 1) % 4
		m.updateInputFocus()
	case "shift+tab", "up":
		m.focusedInput = (m.focusedInput - 1 + 4) % 4
		m.updateInputFocus()
	case "ctrl+l":
		return m.listCollections()
	case "ctrl+f":
		m.state = BackupFormatSelectState
		return m, nil
	case "enter":
		if m.isFormValid() {
			m.state = BackupFormatSelectState
			return m, nil
		}
	}

	switch m.focusedInput {
	case 0:
		m.dbURIInput, cmd = m.dbURIInput.Update(msg)
	case 1:
		m.dbNameInput, cmd = m.dbNameInput.Update(msg)
	case 2:
		m.collectionInput, cmd = m.collectionInput.Update(msg)
	case 3:
		m.outputFileInput, cmd = m.outputFileInput.Update(msg)
	}

	return m, cmd
}

func (m *BackupModel) updateCollectionSelectState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedColl > 0 {
			m.selectedColl--
		}
	case "down", "j":
		if m.selectedColl < len(m.collections)-1 {
			m.selectedColl++
		}
	case "enter":
		if len(m.collections) > 0 {
			m.collectionInput.SetValue(m.collections[m.selectedColl])
			m.state = BackupInputState
			m.showingColls = false
		}
	case "esc":
		m.state = BackupInputState
		m.showingColls = false
	}
	return m, nil
}

func (m *BackupModel) updateFormatSelectState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		return m.startBackup()
	case "esc":
		m.state = BackupInputState
	}
	return m, nil
}

func (m *BackupModel) listCollections() (tea.Model, tea.Cmd) {
	dbURI := strings.TrimSpace(m.dbURIInput.Value())
	dbName := strings.TrimSpace(m.dbNameInput.Value())

	db, err := database.NewMongoDB(dbURI, dbName)
	if err != nil {
		return m, ShowError(err)
	}
	defer db.Close()

	collections, err := db.ListCollections()
	if err != nil {
		return m, ShowError(err)
	}

	m.collections = collections
	m.selectedColl = 0
	m.state = BackupCollectionSelectState
	m.showingColls = true
	return m, nil
}

func (m *BackupModel) updateInputFocus() {
	inputs := []*textinput.Model{&m.dbURIInput, &m.dbNameInput, &m.collectionInput, &m.outputFileInput}
	for i, input := range inputs {
		if i == m.focusedInput {
			input.Focus()
		} else {
			input.Blur()
		}
	}
}

func (m *BackupModel) isFormValid() bool {
	return strings.TrimSpace(m.dbURIInput.Value()) != "" &&
		strings.TrimSpace(m.dbNameInput.Value()) != "" &&
		strings.TrimSpace(m.collectionInput.Value()) != "" &&
		strings.TrimSpace(m.outputFileInput.Value()) != ""
}

func (m *BackupModel) startBackup() (tea.Model, tea.Cmd) {
	m.state = BackupProgressState
	m.backing = true
	m.progressVal = 0
	return m, m.performBackup()
}

func (m *BackupModel) performBackup() tea.Cmd {
	return func() tea.Msg {
		dbURI := strings.TrimSpace(m.dbURIInput.Value())
		dbName := strings.TrimSpace(m.dbNameInput.Value())
		collection := strings.TrimSpace(m.collectionInput.Value())
		outputFile := strings.TrimSpace(m.outputFileInput.Value())
		format := strings.ToLower(m.formats[m.formatSelection])

		result := BackupResult{FilePath: outputFile}

		// Connect to database
		db, err := database.NewMongoDB(dbURI, dbName)
		if err != nil {
			result.Error = fmt.Errorf("failed to connect to MongoDB: %w", err)
			return BackupCompleteMsg{Result: result}
		}
		defer db.Close()

		// Create output file
		file, err := os.Create(outputFile)
		if err != nil {
			result.Error = fmt.Errorf("failed to create output file: %w", err)
			return BackupCompleteMsg{Result: result}
		}
		defer file.Close()

		// Perform backup
		err = db.BackupCollection(collection, file, format)
		if err != nil {
			result.Error = fmt.Errorf("backup failed: %w", err)
			return BackupCompleteMsg{Result: result}
		}

		// Get file info for document count (simplified)
		fileInfo, _ := file.Stat()
		result.DocumentCount = int(fileInfo.Size() / 100) // Rough estimate

		return BackupCompleteMsg{Result: result}
	}
}

func (m *BackupModel) reset() {
	m.state = BackupInputState
	m.backing = false
	m.completed = false
	m.progressVal = 0
	m.result = BackupResult{}
	m.updateInputFocus()
}

func (m *BackupModel) View() string {
	switch m.state {
	case BackupInputState:
		return m.renderInputForm()
	case BackupCollectionSelectState:
		return m.renderCollectionSelector()
	case BackupFormatSelectState:
		return m.renderFormatSelector()
	case BackupProgressState:
		return m.renderProgress()
	case BackupResultState:
		return m.renderResult()
	}
	return ""
}

func (m *BackupModel) renderInputForm() string {
	title := titleStyle.Render("💾 Backup MongoDB Collection")

	form := formStyle.Render(
		labelStyle.Render("Database URI:") + "\n" + m.dbURIInput.View() + "\n\n" +
		labelStyle.Render("Database Name:") + "\n" + m.dbNameInput.View() + "\n\n" +
		labelStyle.Render("Collection:") + "\n" + m.collectionInput.View() + "\n\n" +
		labelStyle.Render("Output File:") + "\n" + m.outputFileInput.View(),
	)

	help := helpStyle.Render("Tab/Shift+Tab: Navigate • Ctrl+L: List collections • Enter: Continue • Esc: Back to menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, form, help)
}

func (m *BackupModel) renderCollectionSelector() string {
	title := titleStyle.Render("📋 Select Collection")

	if len(m.collections) == 0 {
		content := warningStyle.Render("No collections found in database")
		help := helpStyle.Render("Esc: Back to form")
		return lipgloss.JoinVertical(lipgloss.Left, title, content, help)
	}

	var collList string
	for i, coll := range m.collections {
		cursor := " "
		style := menuItemStyle
		if i == m.selectedColl {
			cursor = ">"
			style = selectedMenuItemStyle
		}
		collList += fmt.Sprintf("%s %s\n", cursor, style.Render(coll))
	}

	help := helpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Cancel")

	return lipgloss.JoinVertical(lipgloss.Left, title, collList, help)
}

func (m *BackupModel) renderFormatSelector() string {
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

	help := helpStyle.Render("↑/↓: Navigate • Enter: Start backup • Esc: Back")

	return lipgloss.JoinVertical(lipgloss.Left, title, formatList, help)
}

func (m *BackupModel) renderProgress() string {
	title := titleStyle.Render("💾 Creating Backup...")
	
	progressBar := m.progress.ViewAs(m.progressVal)
	progressText := fmt.Sprintf("Progress: %.1f%%", m.progressVal*100)
	
	content := progressStyle.Render(progressBar + "\n" + progressText)
	help := helpStyle.Render("Please wait while backup is being created...")

	return lipgloss.JoinVertical(lipgloss.Left, title, content, help)
}

func (m *BackupModel) renderResult() string {
	title := titleStyle.Render("💾 Backup Complete")

	var status string
	if m.result.Error != nil {
		status = errorStyle.Render(fmt.Sprintf("❌ Backup failed: %v", m.result.Error))
	} else {
		status = successStyle.Render("✅ Backup completed successfully!")
	}

	stats := fmt.Sprintf(
		"📊 Backup Information:\n"+
		"   Output file: %s\n"+
		"   Format: %s\n"+
		"   Estimated documents: %d",
		m.result.FilePath,
		m.formats[m.formatSelection],
		m.result.DocumentCount,
	)

	help := helpStyle.Render("Enter: Create another backup • Esc: Back to menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, status, stats, help)
}