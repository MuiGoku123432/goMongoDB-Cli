package cmd

import (
	"fmt"
	"log"

	"excelDisclaimer/internal/csv"
	"excelDisclaimer/internal/database"

	"github.com/spf13/cobra"
)

var (
	csvFile    string
	dbURI      string
	dbName     string
	collection string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import CSV data to MongoDB",
	Long:  "Import CSV data to MongoDB collection with Number field matching",
	RunE:  runImport,
}

func init() {
	importCmd.Flags().StringVarP(&csvFile, "csv", "c", "", "CSV file to import (required)")
	importCmd.Flags().StringVarP(&dbURI, "db-uri", "u", "mongodb://localhost:27017", "MongoDB connection URI")
	importCmd.Flags().StringVarP(&dbName, "database", "d", "csvprocessor", "Database name")
	importCmd.Flags().StringVarP(&collection, "collection", "t", "records", "Collection name")
	
	importCmd.MarkFlagRequired("csv")
}

func runImport(cmd *cobra.Command, args []string) error {
	parser := csv.NewParser(csvFile)
	records, err := parser.ParseRecords()
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	log.Printf("Parsed %d records from %s", len(records), csvFile)

	db, err := database.NewMongoDB(dbURI, dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer db.Close()

	successCount := 0
	for i, record := range records {
		if record.Number == "" {
			log.Printf("Skipping record %d: empty Number field", i+1)
			continue
		}

		if err := db.InsertRecord(collection, record); err != nil {
			log.Printf("Failed to insert record %d (Number: %s): %v", i+1, record.Number, err)
			continue
		}
		successCount++
	}

	log.Printf("Successfully imported %d/%d records to %s.%s", successCount, len(records), dbName, collection)
	return nil
}