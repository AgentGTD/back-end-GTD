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
    "github.com/paul-mannino/go-fuzzywuzzy"
    "strings"


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
    date := time.Now()
    dayAndDate := fmt.Sprintf("%s, %s", date.Weekday(), date)
    systemPrompt = fmt.Sprintf("Today is %s ", dayAndDate, systemPrompt)
    reqBody := GroqChatRequest{
        Model: "llama-3.1-8b-instant",
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


// Unified AI Assistant Endpoint

type AIAssistantRequest struct {
    Prompt        string `json:"prompt"`
    Authorization string `header:"Authorization"`
}

type AIAssistantResponse struct {
    Intent      string  `json:"intent"`
    Message     string  `json:"message,omitempty"`
    Task        *Task   `json:"task,omitempty"`
    Project     *Project `json:"project,omitempty"`
    Tasks       []Task  `json:"tasks,omitempty"`
    Summary     string  `json:"summary,omitempty"`
}

// encore:api public method=POST path=/api/ai/assistant
func AIAssistant(ctx context.Context, req *AIAssistantRequest) (*AIAssistantResponse, error) {
    // 1. Parse intent
    parseResp, err := AIParseIntent(ctx, &AIParseIntentRequest{
        Prompt: req.Prompt,
    })
    if err != nil {
        return nil, err
    }

    switch parseResp.Intent {
    case "chat":
        chatResp, err := AIChat(ctx, &AIChatRequest{
            Prompt: req.Prompt,
        })
        if err != nil {
            return nil, err
        }
        return &AIAssistantResponse{
            Intent:  "chat",
            Message: chatResp.Response,
        }, nil

    case "summarize":
        sumResp, err := AISummarize(ctx, &AISummarizeRequest{
            Context: parseResp.Context,
        })
        if err != nil {
            return nil, err
        }
        return &AIAssistantResponse{
            Intent:  "summarize",
            Summary: sumResp.Summary,
            Message: "Here is your summary.",
        }, nil

    case "createTask":
        taskReq := &AICreateTaskRequest{
            Context:       req.Prompt, // or parseResp.Context if you want to use extracted context
            Authorization: req.Authorization,
        }
        taskResp, err := AICreateTask(ctx, taskReq)
        if err != nil {
            return nil, err
        }
        ack := fmt.Sprintf("Task \"%s\" created successfully.", taskResp.Task.Title)
        return &AIAssistantResponse{
            Intent:  "createTask",
            Task:    &taskResp.Task,
            Message: ack,
        }, nil

    case "createProject":
        projReq := &AICreateProjectRequest{
            Prompt:        req.Prompt,
            Authorization: req.Authorization,
        }
        projResp, err := AICreateProject(ctx, projReq)
        if err != nil {
            return nil, err
        }
        ack := fmt.Sprintf("Project \"%s\" created with %d tasks.", projResp.Project.Name, len(projResp.Tasks))
        return &AIAssistantResponse{
            Intent:  "createProject",
            Project: &projResp.Project,
            Tasks:   projResp.Tasks,
            Message: ack,
        }, nil

    case "completeTask":
        completeReq := &AICompleteRequest{
            Prompt:        req.Prompt,
            Authorization: req.Authorization,
        }
        completeResp, err := AICompleteTask(ctx, completeReq)
        if err != nil {
            return nil, err
        }
        return &AIAssistantResponse{
            Intent:   "completeTask",
            Task:     completeResp.Task,
            Tasks:    completeResp.Tasks,
            Project:  completeResp.Project,
            Message:  completeResp.Message,
        }, nil

    default:
        return &AIAssistantResponse{
            Intent:  parseResp.Intent,
            Message: "Sorry, I couldn't understand your request.",
        }, nil
    }
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
    if err == nil {
        idStr := project.ID.Hex()
        return &idStr, nil
    }
    // Not found in DB: create it
    newProject := Project{
        ID:        primitive.NewObjectID(),
        UserID:    userObjID,
        Name:      name,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        TaskCount: 0,
    }
    _, err = col.InsertOne(context.Background(), newProject)
    if err != nil {
        return nil, fmt.Errorf("failed to create project: %v", err)
    }
    idStr := newProject.ID.Hex()
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
        "context_name": bson.M{"$regex": "^" + name + "$", "$options": "i"},
        "userId": userObjID,
    }
    err = col.FindOne(context.Background(), filter).Decode(&nextAction)
    if err == nil {
        idStr := nextAction.ID.Hex()
        return &idStr, nil
    }
    // Not found in DB: create it
    newNextAction := NextAction{
        ID:          primitive.NewObjectID(),
        UserID:      userObjID,
        ContextName: name,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
        TaskCount:   0,
    }
    _, err = col.InsertOne(context.Background(), newNextAction)
    if err != nil {
        return nil, fmt.Errorf("failed to create next action: %v", err)
    }
    idStr := newNextAction.ID.Hex()
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

    /*
    // Update TaskCount in project
     _, err = GetMongoClient().Database("gtd").Collection("projects").UpdateByID(
         ctx,
         project.ID,
         bson.M{"$set": bson.M{"task_count": len(createdTasks)}},
     )
     if err != nil {
         return nil, errors.New("failed to update project task count")
     }
     
     */

    return &AICreateProjectResponse{
        Project: project,
        Tasks:   createdTasks,
    }, nil
}

func stringPtr(s string) *string {
    return &s
}

// Task completion endpoint (only for the single tasks, not for multiple tasks)

/*
type AICompleteTaskRequest struct {
    Prompt        string `json:"prompt"`
    Authorization string `header:"Authorization"`
}

type AICompleteTaskResponse struct {
    Task    *Task   `json:"task,omitempty"`   
    Tasks   []Task  `json:"tasks,omitempty"`  
    Message string  `json:"message"`
}


// encore:api public method=POST path=/api/ai/complete-task
func AICompleteTask(ctx context.Context, req *AICompleteTaskRequest) (*AICompleteTaskResponse, error) {
    userID, err := getUserIDFromContext(ctx, req.Authorization)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    // Use Groq to extract the task title (and optionally project/context)
    resp, err := callGroqChat(req.Prompt, SystemPromptCompleteTask)
    if err != nil {
        return nil, err
    }
    var aiTask struct {
        Title         string `json:"title"`
        ProjectName   string `json:"projectName"`
        NextActionName string `json:"nextActionName"`
    }
    if err := json.Unmarshal([]byte(resp), &aiTask); err != nil {
        return nil, errors.New("AI response could not be parsed as JSON: " + err.Error())
    }
    if aiTask.Title == "" {
        return &AICompleteTaskResponse{
            Message: "I couldn't determine which task to complete. Please specify the task title precisely.",
        }, nil
    }

    matches, err := findAllRelevantTasks(ctx, userID, aiTask.Title, 50)
if err != nil {
    return &AICompleteTaskResponse{
        Message: "Sorry, something went wrong while searching for your task.",
    }, nil
}
if len(matches) == 0 {
    return &AICompleteTaskResponse{
        Message: fmt.Sprintf("I couldn't find any task matching \"%s\".", aiTask.Title),
    }, nil
}
if len(matches) > 1 {
    // Ask user to clarify
    titles := []string{}
    for _, t := range matches {
        titles = append(titles, t.Title)
    }
    return &AICompleteTaskResponse{
        Message: fmt.Sprintf("I found multiple tasks matching \"%s\": %s. Please specify which one you want to complete.", aiTask.Title, strings.Join(titles, "; ")),
        Tasks:   matches,
    }, nil
}

// Only one match, proceed to complete
foundTask := matches[0]
_, err = CompleteTask(ctx, foundTask.ID.Hex(), &GetTasksRequest{Authorization: req.Authorization})
if err != nil {
    return &AICompleteTaskResponse{
        Message: fmt.Sprintf("I found the task \"%s\" but couldn't mark it as complete.", foundTask.Title),
    }, nil
}
return &AICompleteTaskResponse{
    Task:    &foundTask,
    Message: fmt.Sprintf("Task \"%s\" marked as complete!", foundTask.Title),
}, nil
}



func findAllRelevantTasks(ctx context.Context, userID primitive.ObjectID, title string, threshold int) ([]Task, error) {
    client := GetMongoClient()
    tasksCol := client.Database("gtd").Collection("tasks")
    filter := bson.M{
        "userId":    userID,
        "trashed":   false,
        "completed": false,
    }
    cursor, err := tasksCol.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    // Use fuzzy matching to find relevant tasks
    var matches []Task
    for cursor.Next(ctx) {
        var task Task
        if err := cursor.Decode(&task); err != nil {
            continue
        }
        score := fuzzy.Ratio(strings.ToLower(title), strings.ToLower(task.Title))
        if score >= threshold {
            matches = append(matches, task)
        }
    }
    return matches, nil
}
*/


// AI complete endpoint ( a robust version that can handle tasks, projects, and next actions)
type AICompleteRequest struct {
    Prompt        string `json:"prompt"`
    Authorization string `header:"Authorization"`
}
type AICompleteResponse struct {
    Message     string  `json:"message"`
    Task        *Task   `json:"task,omitempty"`
    Tasks       []Task  `json:"tasks,omitempty"`
    Project     *Project `json:"project,omitempty"`
    NextAction  *NextAction `json:"nextAction,omitempty"`
    Count       int     `json:"count,omitempty"`
}

// encore:api public method=POST path=/api/ai/complete
func AICompleteTask(ctx context.Context, req *AICompleteRequest) (*AICompleteResponse, error) {
    userID, err := getUserIDFromContext(ctx, req.Authorization)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    // Use Groq to extract intentType and relevant fields
    resp, err := callGroqChat(req.Prompt, SystemPromptCompleteTask)
    if err != nil {
        return nil, err
    }
    var aiResp struct {
        IntentType     string `json:"intentType"`
        Title          string `json:"title"`
        ProjectName    string `json:"projectName"`
        NextActionName string `json:"nextActionName"`
    }
    if err := json.Unmarshal([]byte(resp), &aiResp); err != nil {
        return nil, errors.New("AI response could not be parsed as JSON: " + err.Error())
    }

    switch aiResp.IntentType {
    case "task":
        // Build filter for project/nextAction if present
        filter := bson.M{
            "userId":    userID,
            "trashed":   false,
            "completed": false,
        }
        if aiResp.ProjectName != "" {
            projectIDPtr, _ := resolveProjectID(aiResp.ProjectName, userID.Hex())
            if projectIDPtr != nil {
                projectID, _ := primitive.ObjectIDFromHex(*projectIDPtr)
                filter["projectId"] = projectID
            }
        }
        if aiResp.NextActionName != "" {
            nextActionIDPtr, _ := resolveNextActionID(aiResp.NextActionName, userID.Hex())
            if nextActionIDPtr != nil {
                nextActionID, _ := primitive.ObjectIDFromHex(*nextActionIDPtr)
                filter["nextActionId"] = nextActionID
            }
        }
        matches, err := findRelevantTasks(ctx, filter, aiResp.Title, 50)

        if err != nil {
            return &AICompleteResponse{Message: "Error searching for your task."}, nil
        }
        if len(matches) == 0 {
            return &AICompleteResponse{Message: fmt.Sprintf("No task found matching \"%s\".", aiResp.Title)}, nil
        }
        if len(matches) > 1 {
            titles := []string{}
            for _, t := range matches {
                titles = append(titles, t.Title)
            }
            return &AICompleteResponse{
                Message: fmt.Sprintf("Multiple tasks found: %s. Please specify.", strings.Join(titles, "; ")),
                Tasks:   matches,
            }, nil
        }
        foundTask := matches[0]
        _, err = CompleteTask(ctx, foundTask.ID.Hex(), &GetTasksRequest{Authorization: req.Authorization})
        if err != nil {
            return &AICompleteResponse{Message: "Could not mark task as complete."}, nil
        }
        return &AICompleteResponse{
            Message: fmt.Sprintf("Task \"%s\" marked as complete!", foundTask.Title),
            Task:    &foundTask,
        }, nil

    case "project":
        // Use your resolveProjectID and fuzzy matching for project name
        projectIDPtr, _ := resolveProjectID(aiResp.ProjectName, userID.Hex())
        if projectIDPtr == nil {
            return &AICompleteResponse{Message: fmt.Sprintf("No project found matching \"%s\".", aiResp.ProjectName)}, nil
        }
        projectID, _ := primitive.ObjectIDFromHex(*projectIDPtr)
        // Mark all tasks in the project as complete
        client := GetMongoClient()
        tasksCol := client.Database("gtd").Collection("tasks")
        res, err := tasksCol.UpdateMany(ctx, bson.M{
            "userId":    userID,
            "projectId": projectID,
            "completed": false,
            "trashed":   false,
        }, bson.M{"$set": bson.M{"completed": true}})
        if err != nil {
            return &AICompleteResponse{Message: "Error completing project tasks."}, nil
        }
        return &AICompleteResponse{
            Message: fmt.Sprintf("Marked %d tasks as complete in project \"%s\".", res.ModifiedCount, aiResp.ProjectName),
            Count:   int(res.ModifiedCount),
        }, nil

    case "nextAction":
        // Use your resolvenextActionID and fuzzy matching for context name
        nextActionIDPtr, _ := resolveNextActionID(aiResp.NextActionName, userID.Hex())
        if nextActionIDPtr == nil {
            return &AICompleteResponse{Message: fmt.Sprintf("No next action/context found matching \"%s\".", aiResp.NextActionName)}, nil
        }
        nextActionID, _ := primitive.ObjectIDFromHex(*nextActionIDPtr)
        client := GetMongoClient()
        tasksCol := client.Database("gtd").Collection("tasks")
        res, err := tasksCol.UpdateMany(ctx, bson.M{
            "userId":       userID,
            "nextActionId": nextActionID,
            "completed":    false,
            "trashed":      false,
        }, bson.M{"$set": bson.M{"completed": true}})
        if err != nil {
            return &AICompleteResponse{Message: "Error completing next action tasks."}, nil
        }
        return &AICompleteResponse{
            Message: fmt.Sprintf("Marked %d tasks as complete in next action \"%s\".", res.ModifiedCount, aiResp.NextActionName),
            Count:   int(res.ModifiedCount),
        }, nil

    default:
        return &AICompleteResponse{Message: "Sorry, I couldn't understand what you want to complete."}, nil
    }
}

func findRelevantTasks(ctx context.Context, filter bson.M, title string, threshold int) ([]Task, error) {
    client := GetMongoClient()
    tasksCol := client.Database("gtd").Collection("tasks")
    cursor, err := tasksCol.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    var matches []Task
    for cursor.Next(ctx) {
        var task Task
        if err := cursor.Decode(&task); err != nil {
            continue
        }
        score := fuzzy.Ratio(strings.ToLower(title), strings.ToLower(task.Title))
        if score >= threshold {
            matches = append(matches, task)
        }
    }
    return matches, nil
}
