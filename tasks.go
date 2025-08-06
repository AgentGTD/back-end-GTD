package encoreapp

import (
	"context"
	"time"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetTasksRequest for fetching all tasks (Authorization header)
type GetTasksRequest struct {
	Authorization string `header:"Authorization"`
}

type GetTasksResponse struct {
	Tasks []Task `json:"tasks"`
}

// encore:api public method=GET path=/api/tasks
func GetTasks(ctx context.Context, req *GetTasksRequest) (*GetTasksResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	tasksCol := client.Database("gtd").Collection("tasks")
	cur, err := tasksCol.Find(ctx, bson.M{"userId": userID, "trashed": false})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	tasks := []Task{}
	for cur.Next(ctx) {
		var t Task
		if err := cur.Decode(&t); err == nil {
			tasks = append(tasks, t)
		}
	}
	return &GetTasksResponse{Tasks: tasks}, nil
}

// CreateTaskRequest for creating a new task
type CreateTaskRequest struct {
	Authorization string  `header:"Authorization"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	DueDate       *string `json:"dueDate"`
	Priority      int     `json:"priority"`
	Category      string  `json:"category"`
	ProjectID     *string `json:"projectId,omitempty"`
	NextActionID  *string `json:"nextActionId,omitempty"`
	Completed     *bool   `json:"completed,omitempty"`
}

type CreateTaskResponse struct {
	Task Task `json:"task"`
}

// encore:api public method=POST path=/api/tasks
func CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	tasksCol := client.Database("gtd").Collection("tasks")
	projectsCol := client.Database("gtd").Collection("projects")

	var dueDate *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		d, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			d, err = time.Parse("2006-01-02", *req.DueDate)
		}
		if err == nil {
			dueDate = &d
		}
	}
	if dueDate == nil {
		now := time.Now()
		dueDate = &now
	}

	var projectID *primitive.ObjectID
	if req.ProjectID != nil && *req.ProjectID != "" {
		id, err := primitive.ObjectIDFromHex(*req.ProjectID)
		if err == nil {
			projectID = &id
		}
	}

	var nextActionID *primitive.ObjectID
	if req.NextActionID != nil && *req.NextActionID != "" {
		id, err := primitive.ObjectIDFromHex(*req.NextActionID)
		if err == nil {
			nextActionID = &id
		}
	}

	// Enforce category logic: if either projectID or nextActionID is set, not inbox
	if (projectID != nil || nextActionID != nil) && req.Category == "inbox" {
		// If client sent 'inbox' but provided project/nextAction, override to 'projects' or 'nextActions' or both
		if projectID != nil && nextActionID != nil {
			req.Category = "projects & nextActions"
		} else if projectID != nil {
			req.Category = "projects"
		} else if nextActionID != nil {
			req.Category = "nextActions"
		}
	}
	if projectID == nil && nextActionID == nil {
		req.Category = "inbox"
	}

	priority := req.Priority
	if priority < 1 || priority > 5 {
		priority = 99
	}

	task := Task{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		ProjectID:    projectID,
		NextActionID: nextActionID,
		Title:        req.Title,
		Description:  req.Description,
		DueDate:      dueDate,
		Priority:     priority,
		Completed:    false,
		Trashed:      false,
		Category:     req.Category,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	_, err = tasksCol.InsertOne(ctx, task)
	if err != nil {
		return nil, errors.New("failed to create task")
	}

	// Increment project task_count if linked
	if projectID != nil {
		_, _ = projectsCol.UpdateOne(
			ctx,
			bson.M{"_id": projectID},
			bson.M{"$inc": bson.M{"task_count": 1}},
		)
	}

	// Increment nextaction task_count if linked
	if nextActionID != nil {
		nextActionsCol := client.Database("gtd").Collection("nextactions")
		_, _ = nextActionsCol.UpdateOne(
			ctx,
			bson.M{"_id": nextActionID},
			bson.M{"$inc": bson.M{"task_count": 1}},
		)
	}

	return &CreateTaskResponse{Task: task}, nil
}


// GetTaskRequest for fetching a specific task
// encore:api public method=GET path=/api/tasks/:id
func GetTask(ctx context.Context, id string, req *GetTasksRequest) (*CreateTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	tasksCol := client.Database("gtd").Collection("tasks")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid task id")
	}
	var task Task
	err = tasksCol.FindOne(ctx, bson.M{"_id": objID, "userId": userID, "trashed": false}).Decode(&task)
	if err != nil {
		return nil, errors.New("task not found")
	}
	return &CreateTaskResponse{Task: task}, nil
}

// UpdateTaskRequest for updating a task
// encore:api public method=PUT path=/api/tasks/:id
func UpdateTask(ctx context.Context, id string, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	tasksCol := client.Database("gtd").Collection("tasks")
	projectsCol := client.Database("gtd").Collection("projects")
	nextActionsCol := client.Database("gtd").Collection("nextactions")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid task id")
	}
	// Fetch existing task to check ownership and previous project/nextaction
	var existing Task
	err = tasksCol.FindOne(ctx, bson.M{"_id": objID, "userId": userID, "trashed": false}).Decode(&existing)
	if err != nil {
		return nil, errors.New("task not found")
	}
	// Prepare update fields
	update := bson.M{"updatedAt": time.Now()}
	if req.Title != "" {
		update["title"] = req.Title
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	if req.DueDate != nil && *req.DueDate != "" {
		d, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			d, err = time.Parse("2006-01-02", *req.DueDate)
		}
		if err == nil {
			update["dueDate"] = d
		}
	}
	update["priority"] = req.Priority
	var newProjectID *primitive.ObjectID
	if req.ProjectID != nil && *req.ProjectID != "" {
		id, err := primitive.ObjectIDFromHex(*req.ProjectID)
		if err == nil {
			update["projectId"] = id
			newProjectID = &id
		}
	} else {
		update["projectId"] = nil
	}
	var newNextActionID *primitive.ObjectID
	if req.NextActionID != nil && *req.NextActionID != "" {
		id, err := primitive.ObjectIDFromHex(*req.NextActionID)
		if err == nil {
			update["nextActionId"] = id
			newNextActionID = &id
		}
	} else {
		update["nextActionId"] = nil
	}
	if req.Category != "" {
		update["category"] = req.Category
	}
	if req.Completed != nil {
		update["completed"] = *req.Completed
	}
	// Enforce business logic: Inbox only if no project/nextAction
	if (update["projectId"] != nil || update["nextActionId"] != nil) && update["category"] == "Inbox" {
		update["category"] = ""
	}
	// --- Project task_count logic ---
	oldProjectID := existing.ProjectID
	if oldProjectID != nil && (newProjectID == nil || *oldProjectID != *newProjectID) {
		projectsCol.UpdateOne(ctx, bson.M{"_id": oldProjectID}, bson.M{"$inc": bson.M{"task_count": -1}})
	}
	if newProjectID != nil && (oldProjectID == nil || *oldProjectID != *newProjectID) {
		projectsCol.UpdateOne(ctx, bson.M{"_id": newProjectID}, bson.M{"$inc": bson.M{"task_count": 1}})
	}
	// --- NextAction task_count logic ---
	oldNextActionID := existing.NextActionID
	if oldNextActionID != nil && (newNextActionID == nil || *oldNextActionID != *newNextActionID) {
		nextActionsCol.UpdateOne(ctx, bson.M{"_id": oldNextActionID}, bson.M{"$inc": bson.M{"task_count": -1}})
	}
	if newNextActionID != nil && (oldNextActionID == nil || *oldNextActionID != *newNextActionID) {
		nextActionsCol.UpdateOne(ctx, bson.M{"_id": newNextActionID}, bson.M{"$inc": bson.M{"task_count": 1}})
	}
	// --- End task_count logic ---
	_, err = tasksCol.UpdateOne(ctx, bson.M{"_id": objID, "userId": userID}, bson.M{"$set": update})
	if err != nil {
		return nil, errors.New("failed to update task")
	}
	// Return updated task
	var updated Task
	err = tasksCol.FindOne(ctx, bson.M{"_id": objID, "userId": userID}).Decode(&updated)
	if err != nil {
		return nil, errors.New("failed to fetch updated task")
	}
	return &CreateTaskResponse{Task: updated}, nil
}

// CompleteTaskRequest for marking a task as complete
// encore:api public method=POST path=/api/tasks/:id/complete
func CompleteTask(ctx context.Context, id string, req *GetTasksRequest) (*CreateTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	tasksCol := client.Database("gtd").Collection("tasks")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid task id")
	}
	// Only allow if user owns the task
	res, err := tasksCol.UpdateOne(ctx, bson.M{"_id": objID, "userId": userID, "trashed": false}, bson.M{"$set": bson.M{"completed": true, "updatedAt": time.Now()}})
	if err != nil || res.MatchedCount == 0 {
		return nil, errors.New("task not found or not authorized")
	}
	var updated Task
	err = tasksCol.FindOne(ctx, bson.M{"_id": objID, "userId": userID}).Decode(&updated)
	if err != nil {
		return nil, errors.New("failed to fetch updated task")
	}
	return &CreateTaskResponse{Task: updated}, nil
}

// Response for deleting a task


type DeleteTaskResponse struct {
	Success bool `json:"success"`
}

// DeleteTaskRequest for deleting a task (soft delete)
// encore:api public method=DELETE path=/api/tasks/:id
func DeleteTask(ctx context.Context, id string, req *GetTasksRequest) (*DeleteTaskResponse, error) {
	userID, err := getUserObjectIDFromAuth(ctx, req.Authorization)
	if err != nil {
		return nil, errors.New("unauthorized")
	}
	client := GetMongoClient()
	tasksCol := client.Database("gtd").Collection("tasks")
	projectsCol := client.Database("gtd").Collection("projects")
	nextActionsCol := client.Database("gtd").Collection("nextactions")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid task id")
	}
	// Fetch task to get projectID/nextActionID before deleting
	var task Task
	err = tasksCol.FindOne(ctx, bson.M{"_id": objID, "userId": userID, "trashed": false}).Decode(&task)
	if err != nil {
		return nil, errors.New("task not found or not authorized")
	}
	// If task has a project, decrement its task_count
	if task.ProjectID != nil {
		projectsCol.UpdateOne(ctx, bson.M{"_id": task.ProjectID}, bson.M{"$inc": bson.M{"task_count": -1}})
	}
	// If task has a next action, decrement its task_count
	if task.NextActionID != nil {
		nextActionsCol.UpdateOne(ctx, bson.M{"_id": task.NextActionID}, bson.M{"$inc": bson.M{"task_count": -1}})
	}
	res, err := tasksCol.UpdateOne(ctx, bson.M{"_id": objID, "userId": userID, "trashed": false}, bson.M{"$set": bson.M{"trashed": true, "updatedAt": time.Now()}})
	if err != nil || res.MatchedCount == 0 {
		return nil, errors.New("task not found or not authorized")
	}
	return &DeleteTaskResponse{Success: true}, nil
}
