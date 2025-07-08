package encoreapp

import (
	"context"
	"log"
	"sync"
	"time"
	"encore.dev/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	once        sync.Once
)


type AppConfig struct {
    JWTSecret   string `env:"JWT_SECRET,required"`
    MongoDBURI  string `env:"MONGODB_URI,required"`
}

// Then in each file:
var cfg = config.Load[AppConfig]()

// GetMongoClient returns a singleton MongoDB client.
func GetMongoClient() *mongo.Client {
	once.Do(func() {
		uri := cfg.MongoDBURI
		if uri == "" {
			log.Fatal("MONGODB_URI environment variable not set")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			log.Fatalf("MongoDB connection error: %v", err)
		}
		mongoClient = client
	})
	return mongoClient
}