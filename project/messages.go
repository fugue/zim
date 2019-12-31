package project

import "time"

// Message sent to give updates on progress
type Message struct {
	BuildID       string    `json:"build_id"`
	Bucket        string    `json:"bucket"`
	OutputKey     string    `json:"output_key"`
	ResultKey     string    `json:"result_key"`
	SourceKey     string    `json:"source_key"`
	CommitID      string    `json:"commit_id"`
	Component     string    `json:"component"`
	Rule        string    `json:"rule"`
	Time          time.Time `json:"time"`
	Exit          bool      `json:"exit"`
	ResponseQueue string    `json:"response_queue"`
	Output        string    `json:"output"`
	Error         string    `json:"error"`
	Worker        string    `json:"worker"`
	Kind          string    `json:"kind"`
	Text          string    `json:"text"`
	Success       bool      `json:"success"`
}
