package encoreapp

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"firebase.google.com/go/v4/auth"
)

func extractIDToken(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return authHeader
}

func getOrCreateUserFromFirebase(ctx context.Context, fbToken *auth.Token) (*User, error) {
	client, err := GetMongoClient()
	if err != nil {
		return nil, errors.New("database connection failed")
	}
	users := client.Database("gtd").Collection("users")

	// Extract user data from Firebase token claims
	email, _ := fbToken.Claims["email"].(string)
	name, _ := fbToken.Claims["name"].(string)
	picture, _ := fbToken.Claims["picture"].(string)
	fbUID := fbToken.UID

	// Validate required fields
	if fbUID == "" {
		return nil, errors.New("invalid Firebase UID")
	}
	if email == "" {
		return nil, errors.New("email not found in token")
	}

	var user User
	err = users.FindOne(ctx, bson.M{"firebaseUid": fbUID}).Decode(&user)

	if err == mongo.ErrNoDocuments {
		// Create new user
		user = User{
			ID:          primitive.NewObjectID(),
			FirebaseUID: fbUID,
			Email:       email,
			Name:        name,
			Picture:     picture,
			CreatedAt:   time.Now(),
		}

		_, err = users.InsertOne(ctx, user)
		if err != nil {
			log.Printf("Failed to create user: %v", err)
			return nil, errors.New("failed to create user")
		}

		log.Printf("Created new user: %s (%s)", email, fbUID)
	} else if err != nil {
		log.Printf("Database error looking up user: %v", err)
		return nil, errors.New("failed to lookup user")
	} else {
		// Update existing user with latest info from Firebase
		update := bson.M{
			"email":     email,
			"updatedAt": time.Now(),
		}
		if name != "" {
			update["name"] = name
		}
		if picture != "" {
			update["picture"] = picture
		}

		_, err = users.UpdateOne(
			ctx,
			bson.M{"firebaseUid": fbUID},
			bson.M{"$set": update},
		)
		if err != nil {
			log.Printf("Failed to update user: %v", err)
		} else {
			// Update local user object with new data
			user.Email = email
			if name != "" {
				user.Name = name
			}
			if picture != "" {
				user.Picture = picture
			}
		}
	}

	return &user, nil
}

type GetUserResponse struct {
	User User `json:"user"`
}

type GetUserRequest struct {
	Authorization string `header:"Authorization"`
}

// encore:api public method=GET path=/api/auth/user
func GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	// Initialize services on first API call
	InitializeServices()

	idToken := extractIDToken(req.Authorization)
	if idToken == "" {
		return nil, errors.New("missing authorization token")
	}

	fbToken, err := getFirebaseUser(ctx, idToken)
	if err != nil {
		log.Printf("Firebase token verification failed: %v", err)
		return nil, errors.New("unauthorized")
	}

	user, err := getOrCreateUserFromFirebase(ctx, fbToken)
	if err != nil {
		log.Printf("Failed to get/create user: %v", err)
		return nil, err
	}

	return &GetUserResponse{User: *user}, nil
}

// Helper function that returns the MongoDB ObjectID for the authenticated user, or an error if unauthorized.
func getUserObjectIDFromAuth(ctx context.Context, authHeader string) (primitive.ObjectID, error) {
	if authHeader == "" {
		return primitive.NilObjectID, errors.New("missing authorization header")
	}

	idToken := extractIDToken(authHeader)
	if idToken == "" {
		return primitive.NilObjectID, errors.New("missing authorization token")
	}

	fbToken, err := getFirebaseUser(ctx, idToken)
	if err != nil {
		log.Printf("Firebase token verification failed: %v", err)
		return primitive.NilObjectID, errors.New("unauthorized")
	}

	user, err := getOrCreateUserFromFirebase(ctx, fbToken)
	if err != nil {
		log.Printf("Failed to get/create user: %v", err)
		return primitive.NilObjectID, errors.New("unauthorized")
	}

	return user.ID, nil
}
