package encoreapp

import (
	"context"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	once        sync.Once
)



// GetMongoClient returns a singleton MongoDB client.
func GetMongoClient() *mongo.Client {
	once.Do(func() {
		uri := secrets.MONGODB_URI
		log.Printf("Connecting to MongoDB at %s", uri)
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