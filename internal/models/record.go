package models

type Record struct {
	Number string                 `csv:"Number" bson:"number"`
	Data   map[string]interface{} `bson:"data"`
}

type CSVRecord struct {
	Number string `csv:"Number"`
}