package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

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

	var records []models.ProductRecord
	decoder, err := csvutil.NewDecoder(csv.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV decoder: %w", err)
	}
	
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("failed to decode CSV: %w", err)
	}

	// Set default AutoSelect value for all records
	for i := range records {
		records[i].AutoSelect = ""
	}

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