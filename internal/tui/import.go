package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"excelDisclaimer/internal/csv"
	"excelDisclaimer/internal/database"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type ImportModel struct {
	state        ImportState
	csvFileInput textinput.Model
	dbURIInput   textinput.Model
	dbNameInput  textinput.Model
	collInput    textinput.Model
	focusedInput int
	progress     progress.Model
	progressVal  float64
	importing    bool
	completed    bool
	result       ImportResult
	files        []string
	selectedFile int
	showingFiles bool
	width        int
	height       int
}

type ImportState int

const (
	ImportInputState ImportState = iota
	ImportFileSelectState
	ImportProgressState
	ImportResultState
)

type ImportResult struct {
	TotalRecords  int
	NewRecords    int
	UpdatedRecords int
	SkippedRecords int
	FailedRecords int
	Error         error
}

type ImportProgressMsg struct {
	Progress float64
	Status   string
}

type ImportCompleteMsg struct {
	Result ImportResult
}

func NewImportModel() *ImportModel {
	csvInput := textinput.New()
	csvInput.Placeholder = "path/to/file.csv"
	csvInput.Focus()

	dbURIInput := textinput.New()
	dbURIInput.Placeholder = "mongodb://localhost:27017"
	dbURIInput.SetValue("mongodb://localhost:27017")

	dbNameInput := textinput.New()
	dbNameInput.Placeholder = "csvprocessor"
	dbNameInput.SetValue("csvprocessor")

	collInput := textinput.New()
	collInput.Placeholder = "records"
	collInput.SetValue("records")

	progressBar := progress.New(
		progress.WithSolidFill("#00aadd"),
		progress.WithoutPercentage(),
	)
	
	return &ImportModel{
		state:        ImportInputState,
		csvFileInput: csvInput,
		dbURIInput:   dbURIInput,
		dbNameInput:  dbNameInput,
		collInput:    collInput,
		focusedInput: 0,
		progress:     progressBar,
	}
}

func (m *ImportModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ImportModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *ImportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case ImportInputState:
			return m.updateInputState(msg)
		case ImportFileSelectState:
			return m.updateFileSelectState(msg)
		case ImportProgressState:
			// No input during progress
			return m, nil
		case ImportResultState:
			if msg.String() == "enter" || msg.String() == " " {
				m.reset()
				return m, nil
			}
		}

	case ImportProgressMsg:
		m.progressVal = msg.Progress
		return m, nil

	case ImportCompleteMsg:
		m.result = msg.Result
		m.importing = false
		m.completed = true
		m.state = ImportResultState
		return m, nil
	}

	return m, cmd
}

func (m *ImportModel) updateInputState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			return m.startImport()
		}
	}

	switch m.focusedInput {
	case 0:
		m.csvFileInput, cmd = m.csvFileInput.Update(msg)
	case 1:
		m.dbURIInput, cmd = m.dbURIInput.Update(msg)
	case 2:
		m.dbNameInput, cmd = m.dbNameInput.Update(msg)
	case 3:
		m.collInput, cmd = m.collInput.Update(msg)
	}

	return m, cmd
}

func (m *ImportModel) updateFileSelectState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.csvFileInput.SetValue(m.files[m.selectedFile])
			m.state = ImportInputState
			m.showingFiles = false
		}
	case "esc":
		m.state = ImportInputState
		m.showingFiles = false
	}
	return m, nil
}

func (m *ImportModel) browseFiles() (tea.Model, tea.Cmd) {
	cwd, _ := os.Getwd()
	files, err := filepath.Glob(filepath.Join(cwd, "*.csv"))
	if err != nil {
		return m, ShowError(err)
	}

	// Convert to relative paths
	for i, file := range files {
		rel, _ := filepath.Rel(cwd, file)
		files[i] = rel
	}

	m.files = files
	m.selectedFile = 0
	m.state = ImportFileSelectState
	m.showingFiles = true
	return m, nil
}

func (m *ImportModel) updateInputFocus() {
	inputs := []*textinput.Model{&m.csvFileInput, &m.dbURIInput, &m.dbNameInput, &m.collInput}
	for i, input := range inputs {
		if i == m.focusedInput {
			input.Focus()
		} else {
			input.Blur()
		}
	}
}

func (m *ImportModel) isFormValid() bool {
	return strings.TrimSpace(m.csvFileInput.Value()) != "" &&
		strings.TrimSpace(m.dbURIInput.Value()) != "" &&
		strings.TrimSpace(m.dbNameInput.Value()) != "" &&
		strings.TrimSpace(m.collInput.Value()) != ""
}

func (m *ImportModel) startImport() (tea.Model, tea.Cmd) {
	m.state = ImportProgressState
	m.importing = true
	m.progressVal = 0
	return m, m.performImport()
}

func (m *ImportModel) performImport() tea.Cmd {
	return func() tea.Msg {
		csvFile := strings.TrimSpace(m.csvFileInput.Value())
		dbURI := strings.TrimSpace(m.dbURIInput.Value())
		dbName := strings.TrimSpace(m.dbNameInput.Value())
		collection := strings.TrimSpace(m.collInput.Value())

		result := ImportResult{}

		// Parse CSV
		parser := csv.NewParser(csvFile)
		records, err := parser.ParseRecords()
		if err != nil {
			result.Error = fmt.Errorf("failed to parse CSV: %w", err)
			return ImportCompleteMsg{Result: result}
		}

		result.TotalRecords = len(records)

		// Connect to database
		db, err := database.NewMongoDB(dbURI, dbName)
		if err != nil {
			result.Error = fmt.Errorf("failed to connect to MongoDB: %w", err)
			return ImportCompleteMsg{Result: result}
		}
		defer db.Close()

		// Import records
		for i, record := range records {
			if record.Number == "" {
				result.SkippedRecords++
				continue
			}

			wasUpdate, err := db.UpsertRecord(collection, record)
			if err != nil {
				result.FailedRecords++
				continue
			}

			if wasUpdate {
				result.UpdatedRecords++
			} else {
				result.NewRecords++
			}

			// In a real implementation, you'd send progress updates via a channel
			_ = float64(i+1) / float64(len(records))
		}

		return ImportCompleteMsg{Result: result}
	}
}

func (m *ImportModel) reset() {
	m.state = ImportInputState
	m.importing = false
	m.completed = false
	m.progressVal = 0
	m.result = ImportResult{}
	m.csvFileInput.SetValue("")
	m.csvFileInput.Focus()
	m.updateInputFocus()
}

func (m *ImportModel) View() string {
	switch m.state {
	case ImportInputState:
		return m.renderInputForm()
	case ImportFileSelectState:
		return m.renderFileSelector()
	case ImportProgressState:
		return m.renderProgress()
	case ImportResultState:
		return m.renderResult()
	}
	return ""
}

func (m *ImportModel) renderInputForm() string {
	adaptiveTitleStyle, adaptiveFormStyle, adaptiveHelpStyle := GetAdaptiveStyles(m.width, m.height)
	
	title := adaptiveTitleStyle.Render("📥 Import CSV to MongoDB")

	form := adaptiveFormStyle.Render(
		labelStyle.Render("CSV File:") + "\n" + m.csvFileInput.View() + "\n\n" +
		labelStyle.Render("Database URI:") + "\n" + m.dbURIInput.View() + "\n\n" +
		labelStyle.Render("Database Name:") + "\n" + m.dbNameInput.View() + "\n\n" +
		labelStyle.Render("Collection:") + "\n" + m.collInput.View(),
	)

	help := adaptiveHelpStyle.Render("Tab/Shift+Tab: Navigate • Ctrl+F: Browse files • Enter: Import • Esc: Back to menu")

	content := lipgloss.JoinVertical(lipgloss.Left, title, form, help)
	
	// Center content if we have space
	if m.width > 0 && m.height > 0 {
		content = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Top,
			content,
		)
	}
	
	return content
}

func (m *ImportModel) renderFileSelector() string {
	title := titleStyle.Render("📁 Select CSV File")

	if len(m.files) == 0 {
		content := warningStyle.Render("No CSV files found in current directory")
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

func (m *ImportModel) renderProgress() string {
	adaptiveTitleStyle, _, adaptiveHelpStyle := GetAdaptiveStyles(m.width, m.height)
	
	title := adaptiveTitleStyle.Render("📥 Importing CSV Data...")
	
	// Make progress bar width responsive
	progressWidth := m.width - 10
	if progressWidth < 20 {
		progressWidth = 20
	}
	if progressWidth > 80 {
		progressWidth = 80
	}
	
	// Create a styled progress bar with adaptive width
	progressBar := lipgloss.NewStyle().Width(progressWidth).Render(m.progress.ViewAs(m.progressVal))
	progressText := fmt.Sprintf("Progress: %.1f%%", m.progressVal*100)
	
	content := progressStyle.Render(progressBar + "\n" + progressText)
	help := adaptiveHelpStyle.Render("Please wait while data is being imported...")

	result := lipgloss.JoinVertical(lipgloss.Left, title, content, help)
	
	// Center content
	if m.width > 0 && m.height > 0 {
		result = lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			result,
		)
	}
	
	return result
}

func (m *ImportModel) renderResult() string {
	title := titleStyle.Render("📥 Import Complete")

	var status string
	if m.result.Error != nil {
		status = errorStyle.Render(fmt.Sprintf("❌ Import failed: %v", m.result.Error))
	} else {
		status = successStyle.Render("✅ Import completed successfully!")
	}

	stats := fmt.Sprintf(
		"📊 Import Statistics:\n"+
		"   Total records: %d\n"+
		"   New records: %d\n"+
		"   Updated records: %d\n"+
		"   Skipped records: %d\n"+
		"   Failed records: %d",
		m.result.TotalRecords,
		m.result.NewRecords,
		m.result.UpdatedRecords,
		m.result.SkippedRecords,
		m.result.FailedRecords,
	)

	help := helpStyle.Render("Enter: Import another file • Esc: Back to menu")

	return lipgloss.JoinVertical(lipgloss.Left, title, status, stats, help)
}