package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"excelDisclaimer/internal/database"
)

type Service struct {
	db *database.MongoDB
}

func NewService(db *database.MongoDB) *Service {
	return &Service{db: db}
}

func (s *Service) BackupCollection(collectionName, outputDir, format string) (string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	extension := "bson"
	if format == "json" {
		extension = "json"
	}

	filename := fmt.Sprintf("backup_%s_%s.%s", collectionName, timestamp, extension)
	filepath := filepath.Join(outputDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	if err := s.db.BackupCollection(collectionName, file, format); err != nil {
		os.Remove(filepath)
		return "", fmt.Errorf("backup failed: %w", err)
	}

	return filepath, nil
}

func (s *Service) BackupDatabase(outputDir, format string) ([]string, error) {
	collections, err := s.db.ListCollections()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	if len(collections) == 0 {
		return nil, fmt.Errorf("no collections found in database")
	}

	var backupFiles []string
	for _, collection := range collections {
		if collection == "system.indexes" {
			continue
		}

		backupFile, err := s.BackupCollection(collection, outputDir, format)
		if err != nil {
			return backupFiles, fmt.Errorf("failed to backup collection %s: %w", collection, err)
		}
		backupFiles = append(backupFiles, backupFile)
	}

	return backupFiles, nil
}

func (s *Service) RestoreCollection(collectionName, inputFile, format string, dropExisting bool) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	if err := s.db.RestoreCollection(collectionName, file, format, dropExisting); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}

func (s *Service) ValidateBackupFile(filename, expectedFormat string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open backup file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot get file info: %w", err)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}

	extension := filepath.Ext(filename)
	if expectedFormat == "json" && extension != ".json" {
		return fmt.Errorf("expected JSON file but got %s", extension)
	}
	if expectedFormat == "bson" && extension != ".bson" {
		return fmt.Errorf("expected BSON file but got %s", extension)
	}

	return nil
}