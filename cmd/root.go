package cmd

import (
	"log"
	"os"

	"excelDisclaimer/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "csv-processor",
	Short: "Interactive MongoDB CSV processor with TUI and CLI modes",
	Long: `MongoDB CSV Processor with both interactive TUI and command-line interfaces.

🖥️  INTERACTIVE MODE (Default):
   Run without arguments to launch the interactive Terminal User Interface (TUI)
   
📄 CLI MODE:
   Use specific commands for scripting and automation:
   • import   - Import CSV files to MongoDB
   • backup   - Backup MongoDB collections  
   • restore  - Restore MongoDB collections
   • tui      - Explicitly launch TUI mode

Examples:
   ./csv-processor                    # Launch interactive TUI (default)
   ./csv-processor import --help      # CLI import help
   ./csv-processor backup --help      # CLI backup help`,
	RunE: runDefaultTUI,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runDefaultTUI(cmd *cobra.Command, args []string) error {
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

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}

func initConfig() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	if os.Getenv("DB_URI") != "" {
		dbURI = os.Getenv("DB_URI")
	}
	if os.Getenv("DB_NAME") != "" {
		dbName = os.Getenv("DB_NAME")
	}
	if os.Getenv("DB_COLLECTION") != "" {
		collection = os.Getenv("DB_COLLECTION")
	}
}