package encoreapp

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirebaseUID  string             `bson:"firebaseUid" json:"firebaseUid"`
	Email        string             `bson:"email" json:"email"`
	Name         string             `bson:"name" json:"name"`
	Picture      string             `bson:"picture,omitempty" json:"picture,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

// Project represents a project owned by a user.
type Project struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	Name        string             `bson:"name" json:"name"`
	Description *string            `bson:"description,omitempty" json:"description,omitempty"`
	TaskCount   int                `bson:"task_count" json:"task_count"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}



// Task represents a task in the system.
type Task struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID  `bson:"userId" json:"userId"`
	ProjectID     *primitive.ObjectID `bson:"projectId" json:"projectId"`
	NextActionID  *primitive.ObjectID `bson:"nextActionId," json:"nextActionId"`
	Title         string              `bson:"title" json:"title"`
	Description   string              `bson:"description" json:"description"`
	DueDate       *time.Time          `bson:"dueDate,omitempty" json:"dueDate,omitempty"`
	Priority      int                 `bson:"priority" json:"priority"`
	Completed     bool                `bson:"completed" json:"completed"`
	Trashed       bool                `bson:"trashed" json:"trashed"`
	Category      string              `bson:"category" json:"category"`
	CreatedAt     time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time           `bson:"updatedAt" json:"updatedAt"`
}


// NextAction represents a next action in the system.
type NextAction struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID  `bson:"userId" json:"userId"`
	ContextName string              `bson:"context_name" json:"context_name"`
	TaskCount   int                 `bson:"task_count" json:"task_count"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt"`
}
