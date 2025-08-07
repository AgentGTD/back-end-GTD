package encoreapp

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	once        sync.Once
	initError   error
)

// GetMongoClient returns a singleton MongoDB client.
func GetMongoClient() (*mongo.Client, error) {
	once.Do(func() {
		uri := secrets.MONGODB_URI
		log.Printf("Connecting to MongoDB at %s", uri)
		if uri == "" {
			initError = errors.New("MONGODB_URI environment variable not set")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			initError = err
			return
		}
		mongoClient = client
		log.Println("MongoDB connected successfully")
	})
	return mongoClient, initError
}
