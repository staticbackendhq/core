package model

import (
	"time"
)

// ExecData represents a server-side function with its name, code and execution
// history
type ExecData struct {
	ID           string        `json:"id"`
	AccountID    string        `json:"accountId"`
	FunctionName string        `json:"name"`
	TriggerTopic string        `json:"trigger"`
	Code         string        `json:"code"`
	Version      int           `json:"version"`
	LastUpdated  time.Time     `json:"lastUpdated"`
	LastRun      time.Time     `json:"lastRun"`
	History      []ExecHistory `json:"history"`
}

// ExecHistory represents a function run ending result
type ExecHistory struct {
	ID         string    `json:"id"`
	FunctionID string    `json:"functionId"`
	Version    int       `json:"version"`
	Started    time.Time `json:"started"`
	Completed  time.Time `json:"completed"`
	Success    bool      `json:"success"`
	Output     []string  `json:"output"`
}

const (
	TaskTypeFunction = "function"
	TaskTypeMessage  = "message"
)

type Task struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      ` json:"type"`
	Value    string      ` json:"value"`
	Meta     interface{} ` json:"meta"`
	Interval string      ` json:"interval"`
	LastRun  time.Time   ` json:"last"`

	BaseName string `json:"base"`
}

type MetaMessage struct {
	Data    string `json:"data"`
	Channel string `json:"channel"`
}
