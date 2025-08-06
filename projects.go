package encoreapp

import (
	"context"
	"time"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Request/response types for projects

type GetProjectsRequest struct {
	Authorization string `header:"Authorization"`
}

type GetProjectsResponse struct {
	Projects []Project `json:"projects"`
}

type CreateProjectRequest struct {
	Authorization string `header:"Authorization"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
}

type CreateProjectResponse struct {
	Project Project `json:"project"`
}

// encore:api public method=GET path=/api/projects
func GetProjects(ctx context.Context, req *GetProjectsRequest) (*GetProjectsResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("projects")
	cur, err := col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	projects := []Project{}
	for cur.Next(ctx) {
		var p Project
		if err := cur.Decode(&p); err == nil {
			projects = append(projects, p)
		}
	}
	return &GetProjectsResponse{Projects: projects}, nil
}

// encore:api public method=POST path=/api/projects
func CreateProject(ctx context.Context, req *CreateProjectRequest) (*CreateProjectResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("projects")
	var descPtr *string
	if req.Description != "" {
		descPtr = &req.Description
	}
	project := Project{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		Name:        req.Name,
		Description: descPtr,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	_, err = col.InsertOne(ctx, project)
	if err != nil {
		return nil, errors.New("failed to create project")
	}
	return &CreateProjectResponse{Project: project}, nil
}

// encore:api public method=GET path=/api/projects/:id
func GetProject(ctx context.Context, id string, req *GetProjectsRequest) (*CreateProjectResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("projects")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid project id")
	}
	var project Project
	err = col.FindOne(ctx, bson.M{"_id": objID, "userId": userID}).Decode(&project)
	if err != nil {
		return nil, errors.New("project not found")
	}
	return &CreateProjectResponse{Project: project}, nil
}

// encore:api public method=PUT path=/api/projects/:id
func UpdateProject(ctx context.Context, id string, req *CreateProjectRequest) (*CreateProjectResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("projects")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid project id")
	}
	update := bson.M{"updatedAt": time.Now()}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	_, err = col.UpdateOne(ctx, bson.M{"_id": objID, "userId": userID}, bson.M{"$set": update})
	if err != nil {
		return nil, errors.New("failed to update project")
	}
	var updated Project
	err = col.FindOne(ctx, bson.M{"_id": objID, "userId": userID}).Decode(&updated)
	if err != nil {
		return nil, errors.New("failed to fetch updated project")
	}
	return &CreateProjectResponse{Project: updated}, nil
}

// encore:api public method=DELETE path=/api/projects/:id
func DeleteProject(ctx context.Context, id string, req *GetProjectsRequest) (*DeleteTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	col := client.Database("gtd").Collection("projects")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid project id")
	}
	res, err := col.DeleteOne(ctx, bson.M{"_id": objID, "userId": userID})
	if err != nil || res.DeletedCount == 0 {
		return nil, errors.New("project not found or not authorized")
	}
	return &DeleteTaskResponse{Success: true}, nil
}
