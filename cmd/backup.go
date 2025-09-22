package cmd

import (
	"fmt"
	"log"

	"excelDisclaimer/internal/backup"
	"excelDisclaimer/internal/database"

	"github.com/spf13/cobra"
)

var (
	outputDir        string
	backupFormat     string
	backupCollection string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup MongoDB collections",
	Long:  "Backup MongoDB collections to BSON or JSON files",
	RunE:  runBackup,
}

func init() {
	backupCmd.Flags().StringVarP(&outputDir, "output", "o", "./backups", "Output directory for backup files")
	backupCmd.Flags().StringVarP(&backupFormat, "format", "f", "bson", "Backup format: bson or json")
	backupCmd.Flags().StringVarP(&backupCollection, "collection", "c", "", "Specific collection to backup (if empty, backs up all collections)")
	backupCmd.Flags().StringVarP(&dbURI, "db-uri", "u", "mongodb://localhost:27017", "MongoDB connection URI")
	backupCmd.Flags().StringVarP(&dbName, "database", "d", "csvprocessor", "Database name")
}

func runBackup(cmd *cobra.Command, args []string) error {
	if backupFormat != "bson" && backupFormat != "json" {
		return fmt.Errorf("invalid format: %s. Use 'bson' or 'json'", backupFormat)
	}

	db, err := database.NewMongoDB(dbURI, dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer db.Close()

	backupService := backup.NewService(db)

	if backupCollection != "" {
		log.Printf("Starting backup of collection '%s' to %s format...", backupCollection, backupFormat)
		backupFile, err := backupService.BackupCollection(backupCollection, outputDir, backupFormat)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		log.Printf("Backup completed successfully: %s", backupFile)
	} else {
		log.Printf("Starting backup of all collections in database '%s' to %s format...", dbName, backupFormat)
		backupFiles, err := backupService.BackupDatabase(outputDir, backupFormat)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		
		log.Printf("Backup completed successfully. Created %d backup files:", len(backupFiles))
		for _, file := range backupFiles {
			log.Printf("  - %s", file)
		}
	}

	return nil
}