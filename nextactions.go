package encoreapp

import (
	"context"
	"time"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Request/response types for next actions

type GetNextActionsRequest struct {
	Authorization string `header:"Authorization"`
}

type GetNextActionsResponse struct {
	NextActions []NextAction `json:"nextActions"`
}

type CreateNextActionRequest struct {
	Authorization string `header:"Authorization"`
	ContextName  string  `json:"context_name"`
}

type CreateNextActionResponse struct {
	NextAction NextAction `json:"nextAction"`
}

// encore:api public method=GET path=/api/next-actions
func GetNextActions(ctx context.Context, req *GetNextActionsRequest) (*GetNextActionsResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("nextactions")
	cur, err := col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	nextActions := []NextAction{}
	for cur.Next(ctx) {
		var na NextAction
		if err := cur.Decode(&na); err == nil {
			nextActions = append(nextActions, na)
		}
	}
	return &GetNextActionsResponse{NextActions: nextActions}, nil
}

// encore:api public method=POST path=/api/next-actions
func CreateNextAction(ctx context.Context, req *CreateNextActionRequest) (*CreateNextActionResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("nextactions")
	nextAction := NextAction{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		ContextName: req.ContextName,
		TaskCount:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err = col.InsertOne(ctx, nextAction)
	if err != nil {
		return nil, errors.New("failed to create next action")
	}
	return &CreateNextActionResponse{NextAction: nextAction}, nil
}

// encore:api public method=GET path=/api/next-actions/:id
func GetNextAction(ctx context.Context, id string, req *GetNextActionsRequest) (*CreateNextActionResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("nextactions")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid next action id")
	}
	var nextAction NextAction
	err = col.FindOne(ctx, bson.M{"_id": objID, "userId": userID}).Decode(&nextAction)
	if err != nil {
		return nil, errors.New("next action not found")
	}
	return &CreateNextActionResponse{NextAction: nextAction}, nil
}

// encore:api public method=PUT path=/api/next-actions/:id
func UpdateNextAction(ctx context.Context, id string, req *CreateNextActionRequest) (*CreateNextActionResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("nextactions")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid next action id")
	}
	update := bson.M{"updatedAt": time.Now()}
	if req.ContextName != "" {
		update["context_name"] = req.ContextName
	}
	_, err = col.UpdateOne(ctx, bson.M{"_id": objID, "userId": userID}, bson.M{"$set": update})
	if err != nil {
		return nil, errors.New("failed to update next action")
	}
	var updated NextAction
	err = col.FindOne(ctx, bson.M{"_id": objID, "userId": userID}).Decode(&updated)
	if err != nil {
		return nil, errors.New("failed to fetch updated next action")
	}
	return &CreateNextActionResponse{NextAction: updated}, nil
}


// encore:api public method=DELETE path=/api/next-actions/:id
func DeleteNextAction(ctx context.Context, id string, req *GetNextActionsRequest) (*DeleteTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("nextactions")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid next action id")
	}
	res, err := col.DeleteOne(ctx, bson.M{"_id": objID, "userId": userID})
	if err != nil || res.DeletedCount == 0 {
		return nil, errors.New("next action not found or not authorized")
	}
	return &DeleteTaskResponse{Success: true}, nil
}
