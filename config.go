package encoreapp

import (
	"log"
	"os"
)

// Centralized configuration secrets for the app
var secrets struct {
	GROQ_API_KEY             string
	MONGODB_URI              string
	FIREBASE_SERVICE_ACCOUNT string
	SERVER_ENV               string
}

// Initialize all services
func InitializeServices() error {
	log.Println("Initializing services...")

	// Initialize secrets from environment variables
	secrets.GROQ_API_KEY = os.Getenv("GROQ_API_KEY")
	secrets.MONGODB_URI = os.Getenv("MONGODB_URI")
	secrets.FIREBASE_SERVICE_ACCOUNT = os.Getenv("FIREBASE_SERVICE_ACCOUNT")
	secrets.SERVER_ENV = os.Getenv("SERVER_ENV")

	if secrets.SERVER_ENV == "" {
		secrets.SERVER_ENV = "development" // Default to development
	}

	// Initialize Firebase (lazy initialization)
	if err := InitFirebase(); err != nil {
		log.Printf("Firebase initialization failed: %v", err)
		// Don't fail startup, just log the error
	}

	// Initialize MongoDB connection (lazy initialization)
	if _, err := GetMongoClient(); err != nil {
		log.Printf("MongoDB initialization failed: %v", err)
		// Don't fail startup, just log the error
	}

	log.Println("Services initialization completed")
	return nil
}
