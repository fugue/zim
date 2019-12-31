package project

import (
	"fmt"
	"time"

	tm "github.com/buger/goterm"
	"github.com/LuminalHQ/zim/queue"
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

type workerStatus struct {
	Name   string
	Status string
}

func printTaskUpdates(
	runID string,
	groups []string,
	workerNames []string,
	queue queue.Queue,
	done chan bool) {

	componentStatus := map[string]string{}
	workerStatus := map[string]string{}
	workerErrors := map[string]string{}
	lineOffset := 3

	colorStatus := func(status string) string {
		switch status {
		case "saving":
			return Cyan(status)
		case "pending":
			return Cyan(status)
		case "error":
			return Red(status)
		case "running":
			return Cyan(status)
		case "done":
			return Green(status)
		}
		return status
	}

	updateState := func(msg *Message) {
		if msg.Error != "" {
			workerErrors[msg.Worker] = msg.Error
		}
		switch msg.Kind {
		case "worker":
			workerStatus[msg.Worker] = msg.Text
		case "rule":
			componentStatus[msg.Component] = msg.Text
		}
	}

	redraw := func() {

		// Clear all text shown on last draw
		tm.Clear()
		tm.MoveCursor(1, 1)

		// Heading at the top
		if len(groups) == 1 {
			tm.Printf("1 worker")
		} else {
			tm.Printf("%d workers", len(groups))
		}

		// Print one line per worker
		for i, group := range groups {
			workerName := workerNames[i]
			tm.MoveCursor(1, i+lineOffset)
			tm.Printf("[%7s] %-12s", colorStatus(workerStatus[workerName]), workerName)
			for k := range group {
				// cName := job.Component.Name()
				cName := ""
				tm.MoveCursor(30+55*k, i+lineOffset)
				tm.Printf("[%7s] %-32s", colorStatus(componentStatus[cName]), cName)
			}
		}

		// Show any errors at the bottom
		tm.MoveCursor(1, len(groups)+lineOffset)
		for workerName, err := range workerErrors {
			tm.Println(workerName, Red(err))
		}
		tm.Flush()
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	redraw()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			redraw()
		default:
			var msg Message
			ok, err := queue.Receive(&msg)
			if ok {
				if msg.BuildID != runID {
					continue // message from a different run of Zim
				}
				updateState(&msg)
				redraw()
			} else if err != nil {
				fmt.Println("Error receiving message:", err)
				time.Sleep(time.Second)
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}
