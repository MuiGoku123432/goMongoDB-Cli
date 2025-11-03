package cmd

import (
	"log"

	"excelDisclaimer/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the interactive TUI for MongoDB operations (same as default)",
	Long: `Start the Terminal User Interface (TUI) for MongoDB operations.
This provides an interactive interface for importing CSV files, 
backing up and restoring MongoDB collections.

Note: This is the same as running the program without any commands.`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	model := tui.NewModel()
	
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running TUI: %v", err)
	}

	return nil
}