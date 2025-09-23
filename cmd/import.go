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

	log.Printf("Parsed %d product records from %s", len(records), csvFile)

	db, err := database.NewMongoDB(dbURI, dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer db.Close()

	successCount := 0
	skippedCount := 0
	for i, record := range records {
		// Validate required fields
		if record.Product == "" && record.Number == "" {
			log.Printf("Skipping record %d: both Product and Number fields are empty", i+1)
			log.Printf("  Raw data: Product='%s', Number='%s', Description='%s'", 
				record.Product, record.Number, record.Description)
			skippedCount++
			continue
		}
		if record.Product == "" {
			log.Printf("Warning: record %d has empty Product field (Number: %s)", i+1, record.Number)
		}
		if record.Number == "" {
			log.Printf("Warning: record %d has empty Number field (Product: %s)", i+1, record.Product)
		}

		if err := db.InsertRecord(collection, record); err != nil {
			log.Printf("Failed to insert record %d (Product: %s, Number: %s): %v", 
				i+1, record.Product, record.Number, err)
			continue
		}
		successCount++
		
		if successCount%100 == 0 {
			log.Printf("Imported %d records...", successCount)
		}
	}
	
	if skippedCount > 0 {
		log.Printf("WARNING: Skipped %d records due to empty fields", skippedCount)
		log.Printf("Check that your CSV column headers match (case-insensitive):")
		log.Printf("  Expected: product, number, description, verbal disclaimer")
	}

	log.Printf("Successfully imported %d/%d records to %s.%s", successCount, len(records), dbName, collection)
	
	if successCount > 0 {
		log.Printf("Sample record structure:")
		log.Printf("  Product: %s", records[0].Product)
		log.Printf("  Number: %s", records[0].Number)
		log.Printf("  Description: %s", records[0].Description)
		log.Printf("  DisclaimerVerbiage: %s", records[0].VerbalDisclaimer)
		log.Printf("  AutoSelect: %s", records[0].AutoSelect)
	}
	
	return nil
}