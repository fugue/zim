package project

// Code indicates the scheduling result for a Rule
type Code int

const (

	// Error indicates the Rule could not be run
	Error Code = iota

	// UpToDate indicates the Rule is up-to-date and doesn't need to be run
	UpToDate

	// ExecError indicates Rule execution was attempted but failed
	ExecError

	// MissingOutputError indicates a Rule output was not produced
	MissingOutputError

	// OK indicates that the Rule executed successfully
	OK

	// Cached indicates the Rule artifact was cached
	Cached
)
