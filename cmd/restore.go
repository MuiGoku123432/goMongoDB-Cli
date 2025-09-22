package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"excelDisclaimer/internal/backup"
	"excelDisclaimer/internal/database"

	"github.com/spf13/cobra"
)

var (
	inputFile        string
	restoreFormat    string
	restoreCollection string
	dropExisting     bool
	skipConfirmation bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore MongoDB collections from backup",
	Long:  "Restore MongoDB collections from BSON or JSON backup files",
	RunE:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input backup file to restore (required)")
	restoreCmd.Flags().StringVarP(&restoreFormat, "format", "f", "", "Backup format: bson or json (auto-detected if not specified)")
	restoreCmd.Flags().StringVarP(&restoreCollection, "collection", "c", "", "Target collection name (defaults to original collection name from backup)")
	restoreCmd.Flags().BoolVar(&dropExisting, "drop", false, "Drop existing collection before restore")
	restoreCmd.Flags().BoolVar(&skipConfirmation, "yes", false, "Skip confirmation prompts")
	restoreCmd.Flags().StringVarP(&dbURI, "db-uri", "u", "mongodb://localhost:27017", "MongoDB connection URI")
	restoreCmd.Flags().StringVarP(&dbName, "database", "d", "csvprocessor", "Database name")
	
	restoreCmd.MarkFlagRequired("input")
}

func runRestore(cmd *cobra.Command, args []string) error {
	if inputFile == "" {
		return fmt.Errorf("input file is required")
	}

	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", inputFile)
	}

	format := restoreFormat
	if format == "" {
		extension := filepath.Ext(inputFile)
		switch extension {
		case ".bson":
			format = "bson"
		case ".json":
			format = "json"
		default:
			return fmt.Errorf("cannot auto-detect format from extension '%s'. Please specify --format", extension)
		}
	}

	if format != "bson" && format != "json" {
		return fmt.Errorf("invalid format: %s. Use 'bson' or 'json'", format)
	}

	targetCollection := restoreCollection
	if targetCollection == "" {
		basename := filepath.Base(inputFile)
		if strings.HasPrefix(basename, "backup_") {
			parts := strings.Split(basename, "_")
			if len(parts) >= 3 {
				targetCollection = parts[1]
			}
		}
		if targetCollection == "" {
			return fmt.Errorf("cannot determine target collection name. Please specify --collection")
		}
	}

	if !skipConfirmation {
		log.Printf("About to restore:")
		log.Printf("  Source file: %s", inputFile)
		log.Printf("  Target database: %s", dbName)
		log.Printf("  Target collection: %s", targetCollection)
		log.Printf("  Format: %s", format)
		if dropExisting {
			log.Printf("  WARNING: Existing collection will be DROPPED!")
		}
		
		if !confirmAction("Do you want to continue?") {
			log.Println("Restore cancelled")
			return nil
		}
	}

	db, err := database.NewMongoDB(dbURI, dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer db.Close()

	backupService := backup.NewService(db)

	if err := backupService.ValidateBackupFile(inputFile, format); err != nil {
		return fmt.Errorf("backup file validation failed: %w", err)
	}

	log.Printf("Starting restore of collection '%s' from %s...", targetCollection, inputFile)
	
	if err := backupService.RestoreCollection(targetCollection, inputFile, format, dropExisting); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	log.Printf("Restore completed successfully!")
	return nil
}

func confirmAction(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}