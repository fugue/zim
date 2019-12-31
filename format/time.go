package format

import "time"

// Unix formats a unix timestamp as a string
func Unix(value int64) string {
	if value == 0 {
		return "-"
	}
	t := time.Unix(value, 0)
	return t.Format(time.RFC3339)
}
