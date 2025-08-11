package encoreapp

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var (
	firebaseApp  *firebase.App
	firebaseAuth *auth.Client
	initOnce     sync.Once
	initErr      error
)

// Call this once at startup (e.g. in main or an init function)
func InitFirebase() error {
	initOnce.Do(func() {
		log.Println("Initializing Firebase...")

		var opt option.ClientOption

		// Use Encore secrets in production, local file in development
		if secrets.SERVER_ENV == "development" {
			opt = option.WithCredentialsJSON([]byte(secrets.FIREBASE_SERVICE_ACCOUNT))
		} else {
			// In production, use the Firebase service account JSON from Encore secrets
			if secrets.FIREBASE_SERVICE_ACCOUNT == "" {
				initError = errors.New("FIREBASE_SERVICE_ACCOUNT secret not set")
				return
			}
			opt = option.WithCredentialsJSON([]byte(secrets.FIREBASE_SERVICE_ACCOUNT))
		}

		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err != nil {
			initError = err
			return
		}
		firebaseApp = app

		// Initialize Firebase Auth with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		firebaseAuth, err = app.Auth(ctx)
		if err != nil {
			initError = err
			return
		}

		log.Println("Firebase initialized successfully")
	})

	return initError
}

// Returns the Firebase user info if the token is valid, else error.
func getFirebaseUser(ctx context.Context, idToken string) (*auth.Token, error) {
	if idToken == "" {
		return nil, errors.New("empty token provided")
	}

	// Initialize Firebase if not already done
	if err := InitFirebase(); err != nil {
		return nil, err
	}

	// Add timeout to token verification
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	token, err := firebaseAuth.VerifyIDToken(verifyCtx, idToken)
	if err != nil {
		log.Printf("Token verification failed: %v", err)
		return nil, errors.New("invalid or expired Firebase ID token")
	}

	// Validate token has required claims
	if token.UID == "" {
		return nil, errors.New("token missing user ID")
	}

	return token, nil
}
