package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"excelDisclaimer/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(uri, dbName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Printf("Connected to MongoDB at %s", uri)

	return &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

func (m *MongoDB) InsertRecord(collectionName string, record interface{}) error {
	collection := m.Database.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("failed to insert record: %w", err)
	}
	return nil
}

// UpsertRecord inserts or updates a record based on Number field
// Preserves existing fields that are not part of the core CSV import
func (m *MongoDB) UpsertRecord(collectionName string, record models.ProductRecord) (bool, error) {
	collection := m.Database.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use Number field as unique identifier
	filter := bson.M{"Number": record.Number}
	
	// First, check if record exists to preserve extra fields
	var existingDoc bson.M
	err := collection.FindOne(ctx, filter).Decode(&existingDoc)
	
	wasUpdate := false
	var finalDoc bson.M
	
	if err == nil {
		// Record exists - preserve extra fields
		wasUpdate = true
		
		// Convert new record to bson.M for merging
		newDocBytes, err := bson.Marshal(record)
		if err != nil {
			return false, fmt.Errorf("failed to marshal new record: %w", err)
		}
		
		var newDoc bson.M
		if err := bson.Unmarshal(newDocBytes, &newDoc); err != nil {
			return false, fmt.Errorf("failed to unmarshal new record: %w", err)
		}
		
		// Start with existing document to preserve all extra fields
		finalDoc = existingDoc
		
		// Update core fields from CSV import
		coreFields := []string{"Product", "Number", "Description", "DisclaimerVerbiage", "AutoSelect"}
		for _, field := range coreFields {
			if value, exists := newDoc[field]; exists {
				finalDoc[field] = value
			}
		}
		
		// Count extra fields (excluding _id and core fields)
		extraFieldCount := 0
		for key := range existingDoc {
			if key != "_id" {
				isCore := false
				for _, coreField := range coreFields {
					if key == coreField {
						isCore = true
						break
					}
				}
				if !isCore {
					extraFieldCount++
				}
			}
		}
		
		log.Printf("Updated existing record with Number: %s (preserved %d extra fields)", 
			record.Number, extraFieldCount)
	} else if err == mongo.ErrNoDocuments {
		// Record doesn't exist - use new record as-is
		newDocBytes, err := bson.Marshal(record)
		if err != nil {
			return false, fmt.Errorf("failed to marshal new record: %w", err)
		}
		
		if err := bson.Unmarshal(newDocBytes, &finalDoc); err != nil {
			return false, fmt.Errorf("failed to unmarshal new record: %w", err)
		}
		
		log.Printf("Inserted new record with Number: %s", record.Number)
	} else {
		return false, fmt.Errorf("failed to check existing record with Number %s: %w", record.Number, err)
	}
	
	// Replace entire document with merged data
	opts := options.Replace().SetUpsert(true)
	_, err = collection.ReplaceOne(ctx, filter, finalDoc, opts)
	if err != nil {
		return false, fmt.Errorf("failed to upsert record with Number %s: %w", record.Number, err)
	}
	
	return wasUpdate, nil
}

func (m *MongoDB) ListCollections() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := m.Database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return cursor, nil
}

func (m *MongoDB) BackupCollection(collectionName string, writer io.Writer, format string) error {
	collection := m.Database.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	count := 0
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return fmt.Errorf("failed to decode document: %w", err)
		}

		var data []byte
		if format == "json" {
			data, err = json.Marshal(doc)
			if err != nil {
				return fmt.Errorf("failed to marshal to JSON: %w", err)
			}
			data = append(data, '\n')
		} else {
			data, err = bson.Marshal(doc)
			if err != nil {
				return fmt.Errorf("failed to marshal to BSON: %w", err)
			}
		}

		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write backup data: %w", err)
		}
		count++

		if count%1000 == 0 {
			log.Printf("Backed up %d documents...", count)
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	log.Printf("Backup completed: %d documents from collection '%s'", count, collectionName)
	return nil
}

func (m *MongoDB) RestoreCollection(collectionName string, reader io.Reader, format string, dropExisting bool) error {
	collection := m.Database.Collection(collectionName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if dropExisting {
		if err := collection.Drop(ctx); err != nil {
			log.Printf("Warning: failed to drop collection %s: %v", collectionName, err)
		}
	}

	var documents []interface{}
	const batchSize = 1000

	if format == "json" {
		decoder := json.NewDecoder(reader)
		for {
			var doc bson.M
			if err := decoder.Decode(&doc); err == io.EOF {
				break
			} else if err != nil {
				return fmt.Errorf("failed to decode JSON: %w", err)
			}
			documents = append(documents, doc)

			if len(documents) >= batchSize {
				if err := m.insertBatch(collection, documents); err != nil {
					return err
				}
				documents = documents[:0]
			}
		}
	} else {
		buffer := make([]byte, 4096)
		var docBuffer []byte

		for {
			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read BSON data: %w", err)
			}

			docBuffer = append(docBuffer, buffer[:n]...)

			for len(docBuffer) >= 4 {
				docSize := int(docBuffer[0]) | int(docBuffer[1])<<8 | int(docBuffer[2])<<16 | int(docBuffer[3])<<24
				if len(docBuffer) < docSize {
					break
				}

				var doc bson.M
				if err := bson.Unmarshal(docBuffer[:docSize], &doc); err != nil {
					return fmt.Errorf("failed to unmarshal BSON: %w", err)
				}

				documents = append(documents, doc)
				docBuffer = docBuffer[docSize:]

				if len(documents) >= batchSize {
					if err := m.insertBatch(collection, documents); err != nil {
						return err
					}
					documents = documents[:0]
				}
			}
		}
	}

	if len(documents) > 0 {
		if err := m.insertBatch(collection, documents); err != nil {
			return err
		}
	}

	log.Printf("Restore completed: imported documents to collection '%s'", collectionName)
	return nil
}

func (m *MongoDB) insertBatch(collection *mongo.Collection, documents []interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := collection.InsertMany(ctx, documents)
	if err != nil {
		return fmt.Errorf("failed to insert batch: %w", err)
	}

	log.Printf("Inserted batch of %d documents", len(documents))
	return nil
}