package project

import (
	"github.com/fatih/color"
)

var (
	// Bright highlights text in the terminal
	Bright func(args ...interface{}) string

	// Cyan text color
	Cyan func(args ...interface{}) string

	// Green text color
	Green func(args ...interface{}) string

	// Red text color
	Red func(args ...interface{}) string
)

func init() {
	Bright = color.New(color.FgHiWhite).SprintFunc()
	Cyan = color.New(color.FgCyan).SprintFunc()
	Green = color.New(color.FgGreen).SprintFunc()
	Red = color.New(color.FgRed).SprintFunc()
}
