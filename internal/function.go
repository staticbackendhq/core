package internal

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExecData represents a server-side function with its name, code and execution
// history
type ExecData struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	AccountID    primitive.ObjectID `bson:"accountId" json:"accountId"`
	FunctionName string             `bson:"name" json:"name"`
	Code         string             `bson:"code" json:"code"`
	Version      int                `bson:"v" json:"version"`
	LastUpdated  time.Time          `bson:"lu" json:"lastUpdated"`
	History      []ExecHistory      `bson:"h" json:"history"`
}

// ExecHistory represents a function run ending result
type ExecHistory struct {
	Version   int       `bson:"v" json:"version"`
	Started   time.Time `bson:"s" json:"started"`
	Completed time.Time `bson:"c" json:"completed"`
	Success   bool      `bson:"ok" json:"success"`
	Output    []string  `bson:"out" json:"output"`
}
