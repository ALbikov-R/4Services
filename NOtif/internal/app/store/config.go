package store

type Config struct {
	MongoURI       string
	DataBaseName   string
	CollectionName string
}

func NewConfig() *Config {
	return &Config{
		MongoURI:       "mongodb://localhost:27017",
		DataBaseName:   "MessageService",
		CollectionName: "Massages",
	}
}
