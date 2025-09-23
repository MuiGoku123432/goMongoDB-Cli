package models

type ProductRecord struct {
	Product           string `csv:"product" bson:"Product"`
	Number            string `csv:"number" bson:"Number"`
	Description       string `csv:"description" bson:"Description"`
	VerbalDisclaimer  string `csv:"verbal disclaimer" bson:"DisclaimerVerbiage"`
	AutoSelect        string `bson:"AutoSelect"`
}

// Legacy Record struct for backward compatibility with existing backup/restore
type Record struct {
	Number string                 `csv:"Number" bson:"number"`
	Data   map[string]interface{} `bson:"data"`
}

type CSVRecord struct {
	Number string `csv:"Number"`
}