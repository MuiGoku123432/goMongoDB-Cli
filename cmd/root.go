package cmd

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "csv-processor",
	Short: "A CLI tool for processing CSV files and importing to MongoDB",
	Long: `CSV Processor is a command-line tool that helps you import CSV files 
into MongoDB collections with support for Number field matching.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
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