package encoreapp

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"


    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type GroqChatRequest struct {
    Messages []GroqMessage `json:"messages"`
    Model    string        `json:"model"`
}

type GroqMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type GroqChatResponse struct {
    Choices []struct {
        Message GroqMessage `json:"message"`
    } `json:"choices"`
}

func callGroqChat(userPrompt string, systemPrompt string) (string, error) {
    today := time.Now().Format("2006-01-02")
    systemPrompt = fmt.Sprintf(systemPrompt, today)
    reqBody := GroqChatRequest{
        Model: "llama3-70b-8192",
        Messages: []GroqMessage{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: userPrompt},
        },
    }
    b, _ := json.Marshal(reqBody)
    req, _ := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(b))
    req.Header.Set("Authorization", "Bearer "+secrets.GROQ_API_KEY)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        return "", errors.New(fmt.Sprintf("Groq API error: %s", string(body)))
    }
    var groqResp GroqChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
        return "", err
    }
    if len(groqResp.Choices) == 0 {
        return "", errors.New("no response from Groq")
    }
    return groqResp.Choices[0].Message.Content, nil
}


// AI Endpoints

// Parse intent endpoint
type AIParseIntentRequest struct {
    Prompt string `json:"prompt"`
}

type AIParseIntentResponse struct {
    Intent      string `json:"intent"`      
    UserPrompt  string `json:"userPrompt"`  
    Context     string `json:"context,omitempty"`
    Title       string `json:"title,omitempty"`
    Description string `json:"description,omitempty"`
    ProjectName string `json:"projectName,omitempty"`
    NextActionName string `json:"nextActionName,omitempty"`
    // other fields can be added as needed
}

// encore:api public method=POST path=/api/ai/parse-intent
func AIParseIntent(ctx context.Context, req *AIParseIntentRequest) (*AIParseIntentResponse, error) {
    resp, err := callGroqChat(req.Prompt, SystemPromptParseIntent)
    if err != nil {
        return nil, err
    }
    var parsed AIParseIntentResponse
    if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
        return nil, errors.New("AI response could not be parsed as JSON: " + err.Error())
    }
    // If projectId/nextActionId are empty, leave them as empty string (or handle as needed)
    return &parsed, nil
}


// General chat endpoint
type AIChatRequest struct {
    Prompt string `json:"prompt"`
}

type AIChatResponse struct {
    Response string `json:"response"`
}

// encore:api public method=POST path=/api/ai/chat
func AIChat(ctx context.Context, req *AIChatRequest) (*AIChatResponse, error) {
    resp, err := callGroqChat(req.Prompt, SystemPromptChat)
    if err != nil {
        return nil, err
    }
    return &AIChatResponse{Response: resp}, nil
}


// Summarization endpoint
type AISummarizeRequest struct {
    Context string `json:"context"`
}
type AISummarizeResponse struct {
    Summary string `json:"summary"`
}

// encore:api public method=POST path=/api/ai/summarize
func AISummarize(ctx context.Context, req *AISummarizeRequest) (*AISummarizeResponse, error) {
    prompt := "Summarize the following context and suggest improvements:\n" + req.Context
    resp, err := callGroqChat(prompt, SystemPromptSummarizer)
    if err != nil {
        return nil, err
    }
    return &AISummarizeResponse{Summary: resp}, nil
}


// Task creation endpoint
type AICreateTaskRequest struct {
    Context       string `json:"context"`
    Authorization string `header:"Authorization"`
}

type AICreateTaskResponse struct {
    Task Task `json:"task"`
}

// encore:api public method=POST path=/api/ai/create-task
func AICreateTask(ctx context.Context, req *AICreateTaskRequest) (*AICreateTaskResponse, error) {
    userID, err := getUserIDFromContext(ctx, req.Authorization)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    prompt := "Create a task for the following objective/context:\n" + req.Context
    resp, err := callGroqChat(prompt, SystemPromptCreateTask)
    fmt.Println("\nAI response:", resp)
    if err != nil {
        return nil, err
    }

    var aiTask struct {
        Title           string `json:"title"`
        Description     string `json:"description"`
        DueDate         string `json:"dueDate"` 
        Priority        int    `json:"priority"`
        Category        string `json:"category"`
        ProjectName     string `json:"projectName"`     
        NextActionName  string `json:"nextActionName"`   
    }
    if err := json.Unmarshal([]byte(resp), &aiTask); err != nil {
        return nil, errors.New("AI response could not be parsed as JSON: " + err.Error())
    }
    
    fmt.Println("\nParsed AI task:", aiTask)

    var projectIDPtr, nextActionIDPtr *string
    if aiTask.ProjectName != "" {
        projectIDPtr, _ = resolveProjectID(aiTask.ProjectName, userID.Hex())
    }
 
     if aiTask.NextActionName != "" {
        nextActionIDPtr, _ = resolveNextActionID(aiTask.NextActionName, userID.Hex())
    }

    dueDateStr := aiTask.DueDate
    if dueDateStr == "" {
        dueDateStr = time.Now().Format(time.RFC3339)
    }

    createReq := &CreateTaskRequest{
        Authorization: req.Authorization,
        Title:         aiTask.Title,
        Description:   aiTask.Description,
        DueDate:       &dueDateStr,
        Priority:      aiTask.Priority,
        Category:      aiTask.Category,
        ProjectID:     projectIDPtr,
        NextActionID:  nextActionIDPtr,
    }

    fmt.Println("\nCreating task with request:", createReq)
    taskResp, err := CreateTask(ctx, createReq)
    if err != nil {
        return nil, err
    }
    return &AICreateTaskResponse{Task: taskResp.Task}, nil
}


func resolveProjectID(name string, userID string) (*string, error) {
    if name == "" {
        return nil, nil
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        return nil, fmt.Errorf("invalid userID format: %v", err)
    }

    client := GetMongoClient()
    col := client.Database("gtd").Collection("projects")
    var project struct{ ID primitive.ObjectID `bson:"_id"` }

    filter := bson.M{
        "name": bson.M{"$regex": "^" + name + "$", "$options": "i"},
        "userId": userObjID,
    }

    err = col.FindOne(context.Background(), filter).Decode(&project)
    if err != nil {
        fmt.Println("Project not found for name:", name)
        return nil, nil
    }

    idStr := project.ID.Hex()
    return &idStr, nil
}


func resolveNextActionID(name string, userID string) (*string, error) {
    
    if name == "" {
        return nil, nil
    }

    userObjID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
        return nil, fmt.Errorf("invalid userID format: %v", err)
    }

    client := GetMongoClient()
    col := client.Database("gtd").Collection("nextactions")
    var nextAction struct{ ID primitive.ObjectID `bson:"_id"` }

    filter := bson.M{
        "context_name":   bson.M{"$regex": "^" + name + "$", "$options": "i"},
        "userId": userObjID,
    }

    err = col.FindOne(context.Background(), filter).Decode(&nextAction)
    if err != nil {
        fmt.Println("\nNext action not found for name:", name)
        return nil, fmt.Errorf("next action not found for name '%s': %v", name, err)
    }

    idStr := nextAction.ID.Hex()
    return &idStr, nil
}



// project creation endpoint
type AICreateProjectRequest struct {
    Prompt        string `json:"prompt"`
    Authorization string `header:"Authorization"`
}

type AICreateProjectResponse struct {
    Project Project   `json:"project"`
    Tasks   []Task    `json:"tasks"`
}

// encore:api public method=POST path=/api/ai/create-project
func AICreateProject(ctx context.Context, req *AICreateProjectRequest) (*AICreateProjectResponse, error) {
    resp, err := callGroqChat(req.Prompt, SystemPromptCreateProject)
    if err != nil {
        return nil, err
    }

    var aiResp struct {
        ProjectName        string `json:"projectName"`
        ProjectDescription string `json:"projectDescription"`
        Tasks              []struct {
            Title       string `json:"title"`
            Description string `json:"description"`
            DueDate     string `json:"dueDate"`
            Priority    int    `json:"priority"`
            Category    string `json:"category"`
        } `json:"tasks"`
    }
    if err := json.Unmarshal([]byte(resp), &aiResp); err != nil {
        return nil, errors.New("AI response could not be parsed as JSON: " + err.Error())
    }

    
    //  Create the project using your existing function
    createProjectReq := &CreateProjectRequest{
        Authorization: req.Authorization,
        Name:          aiResp.ProjectName,
        Description:   aiResp.ProjectDescription, 
    }
    projectResp, err := CreateProject(ctx, createProjectReq)
    if err != nil {
        return nil, err
    }
    project := projectResp.Project

    //  Create tasks using your existing function
    createdTasks := []Task{}
    for _, t := range aiResp.Tasks {
        dueDateStr := t.DueDate
        if dueDateStr == "" {
            dueDateStr = time.Now().Format(time.RFC3339)
        }
        createTaskReq := &CreateTaskRequest{
            Authorization: req.Authorization,
            Title:         t.Title,
            Description:   t.Description,
            DueDate:       &dueDateStr,
            Priority:      t.Priority,
            Category:      t.Category,
            ProjectID:     stringPtr(project.ID.Hex()),
        }
        taskResp, err := CreateTask(ctx, createTaskReq)
        if err == nil {
            createdTasks = append(createdTasks, taskResp.Task)
        }
    }

    // Update TaskCount in project
     _, err = GetMongoClient().Database("gtd").Collection("projects").UpdateByID(
         ctx,
         project.ID,
         bson.M{"$set": bson.M{"task_count": len(createdTasks)}},
     )
     if err != nil {
         return nil, errors.New("failed to update project task count")
     }
     
    return &AICreateProjectResponse{
        Project: project,
        Tasks:   createdTasks,
    }, nil
}

func stringPtr(s string) *string {
    return &s
}

