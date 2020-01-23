package sign

import "time"

// Input for a signing request
type Input struct {
	Method        string            `json:"method"`
	Name          string            `json:"name"`
	Metadata      map[string]string `json:"metadata"`
	ContentLength int64             `json:"content_len"`
}

// Output from a signing request
type Output struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Item contains information about an item in storage
type Item struct {
	Key          string            `json:"key"`
	Metadata     map[string]string `json:"metadata"`
	Version      string            `json:"version"`
	ETag         string            `json:"etag"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"last_modified"`
}
