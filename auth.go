package encoreapp

import (
	"context"
	"os"
	"time"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func init() {
    fmt.Println("JWT_SECRET length:", len(jwtSecret))
}

// SignupRequest is the input for user registration.
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type SignupResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// encore:api public method=POST path=/api/auth/signup
func Signup(ctx context.Context, req *SignupRequest) (*SignupResponse, error) {
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return nil, errors.New("all fields are required")
	}
	client := GetMongoClient()
	users := client.Database("gtd").Collection("users")

	// Check if user exists
	var existing User
	err := users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		return nil, errors.New("user already exists")
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	user := User{
		ID:        primitive.NewObjectID(),
		Email:     req.Email,
		Password:  string(hash),
		Name:      req.Name,
		CreatedAt: time.Now(),
	}
	_, err = users.InsertOne(ctx, user)
	if err != nil {
		return nil, errors.New("failed to create user")
	}

	token, err := generateJWT(user.ID.Hex())
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	user.Password = "" // do not return password
	return &SignupResponse{Token: token, User: user}, nil
}

func generateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"userId": userID,
		"exp":    time.Now().Add(72 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}



// LoginRequest is the input for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// encore:api public method=POST path=/api/auth/login
func Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}
	client := GetMongoClient()
	users := client.Database("gtd").Collection("users")

	var user User
	err := users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := generateJWT(user.ID.Hex())
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	user.Password = ""
	return &LoginResponse{Token: token, User: user}, nil
}



// GetUserResponse is the output for getting the current user.
type GetUserResponse struct {
	User User `json:"user"`
}

// GetUserRequest is the input for getting the current user.
type GetUserRequest struct {
    Authorization string `header:"Authorization"`
}

// encore:api public method=GET path=/api/auth/user
func GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
    userID, err := getUserIDFromContext(ctx, req.Authorization)
    if err != nil {
        return nil, errors.New("unauthorized")
    }
    client := GetMongoClient()
    users := client.Database("gtd").Collection("users")

    var user User
    err = users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
    if err != nil {
        return nil, errors.New("user not found")
    }
    user.Password = ""
    return &GetUserResponse{User: user}, nil
}

// Helper to extract user ID from JWT in context
func getUserIDFromContext(ctx context.Context, Authorization string) (primitive.ObjectID, error) {
    authHeader := Authorization
    // Accept both "Bearer <token>" and just "<token>"
    var tokenStr string
    if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
        tokenStr = authHeader[7:]
    } else {
        tokenStr = authHeader
    }
    token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    })
    if err != nil || !token.Valid {
        return primitive.NilObjectID, errors.New("invalid token")
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return primitive.NilObjectID, errors.New("invalid token claims")
    }
    userIDStr, ok := claims["userId"].(string)
    if !ok {
        return primitive.NilObjectID, errors.New("invalid userId in token")
    }
    return primitive.ObjectIDFromHex(userIDStr)
}
