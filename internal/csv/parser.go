package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"excelDisclaimer/internal/models"

	"github.com/jszwec/csvutil"
)

type Parser struct {
	filename string
}

func NewParser(filename string) *Parser {
	return &Parser{filename: filename}
}

func (p *Parser) ParseRecords() ([]models.ProductRecord, error) {
	file, err := os.Open(p.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// First, inspect the CSV headers
	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	log.Printf("CSV Headers found: %v", headers)
	log.Printf("Expected headers (case-insensitive): [product, number, description, verbal disclaimer]")

	// Reset file to beginning for parsing
	file.Seek(0, 0)

	// Parse using case-insensitive approach
	records, err := p.parseFlexible(file)
	if err != nil {
		return nil, err
	}

	// Set default AutoSelect value for all records
	for i := range records {
		records[i].AutoSelect = ""
	}

	return records, nil
}

func (p *Parser) parseFlexible(file *os.File) ([]models.ProductRecord, error) {
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	// Create a map of lowercase headers to their indices
	headerMap := make(map[string]int)
	for i, header := range headers {
		normalized := strings.TrimSpace(strings.ToLower(header))
		headerMap[normalized] = i
		log.Printf("  Column %d: '%s' (normalized: '%s')", i, header, normalized)
	}

	// Check for required columns
	requiredColumns := map[string]string{
		"product":           "Product",
		"number":            "Number",
		"description":       "Description",
		"verbal disclaimer": "DisclaimerVerbiage",
	}

	missingColumns := []string{}
	for csvName, fieldName := range requiredColumns {
		if _, exists := headerMap[csvName]; !exists {
			missingColumns = append(missingColumns, fmt.Sprintf("%s (for field %s)", csvName, fieldName))
		}
	}

	if len(missingColumns) > 0 {
		log.Printf("WARNING: Missing expected columns: %v", missingColumns)
		log.Printf("This may result in empty fields in the imported data")
	}

	var records []models.ProductRecord
	rowNum := 1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row %d: %w", rowNum+1, err)
		}
		rowNum++

		record := models.ProductRecord{}

		// Map columns flexibly
		if idx, exists := headerMap["product"]; exists && idx < len(row) {
			record.Product = strings.TrimSpace(row[idx])
		}
		if idx, exists := headerMap["number"]; exists && idx < len(row) {
			record.Number = strings.TrimSpace(row[idx])
		}
		if idx, exists := headerMap["description"]; exists && idx < len(row) {
			record.Description = strings.TrimSpace(row[idx])
		}
		if idx, exists := headerMap["verbal disclaimer"]; exists && idx < len(row) {
			record.VerbalDisclaimer = strings.TrimSpace(row[idx])
		}

		records = append(records, record)
	}

	log.Printf("Parsed %d records from CSV", len(records))
	return records, nil
}

// Legacy method for backward compatibility
func (p *Parser) ParseLegacyRecords() ([]models.Record, error) {
	file, err := os.Open(p.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var records []models.Record
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		record := models.Record{
			Data: make(map[string]interface{}),
		}

		for i, value := range row {
			if i < len(headers) {
				if headers[i] == "Number" {
					record.Number = value
				}
				record.Data[headers[i]] = value
			}
		}

		records = append(records, record)
	}

	return records, nil
}

func (p *Parser) ParseWithCSVUtil() ([]models.CSVRecord, error) {
	file, err := os.Open(p.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	var records []models.CSVRecord
	decoder, err := csvutil.NewDecoder(csv.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV decoder: %w", err)
	}
	
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("failed to decode CSV: %w", err)
	}

	return records, nil
}