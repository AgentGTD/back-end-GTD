package encoreapp

import (
    "context"
    "errors"
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
    client := GetMongoClient()
    users := client.Database("gtd").Collection("users")

    email, _ := fbToken.Claims["email"].(string)
    name, _ := fbToken.Claims["name"].(string)
    picture, _ := fbToken.Claims["picture"].(string)
    fbUID := fbToken.UID

    var user User
    err := users.FindOne(ctx, bson.M{"firebaseUid": fbUID}).Decode(&user)
    if err == mongo.ErrNoDocuments {
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
            return nil, errors.New("failed to create user")
        }
    } else if err != nil {
        return nil, errors.New("failed to lookup user")
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
    idToken := extractIDToken(req.Authorization)
    fbToken, err := getFirebaseUser(ctx, idToken)
    if err != nil {
        return nil, errors.New("unauthorized")
    }
    user, err := getOrCreateUserFromFirebase(ctx, fbToken)
    if err != nil {
        return nil, err
    }
    return &GetUserResponse{User: *user}, nil
}


type LogoutRequest struct {
    Authorization string `header:"Authorization"`
}

/*
// encore:api public method=POST path=/api/auth/logout
func Logout(ctx context.Context, req *LogoutRequest) error {
    // With Firebase, logout is handled on the client by deleting the token.
    // Optionally, you can revoke tokens using firebaseAuth.RevokeRefreshTokens(ctx, uid)
    return nil
}
    */


// Helper function that returns the MongoDB ObjectID for the authenticated user, or an error if unauthorized.
func getUserObjectIDFromAuth(ctx context.Context, authHeader string) (primitive.ObjectID, error) {
    idToken := extractIDToken(authHeader)
    fbToken, err := getFirebaseUser(ctx, idToken)
    if err != nil {
        return primitive.NilObjectID, errors.New("unauthorized")
    }
    user, err := getOrCreateUserFromFirebase(ctx, fbToken)
    if err != nil {
        return primitive.NilObjectID, errors.New("unauthorized")
    }
    return user.ID, nil
}

