package project

import (
	"fmt"
	"os"
	"path"
	"sort"

	glob "github.com/bmatcuk/doublestar"
)

// MatchFiles returns files within the directory that match the pattern
func MatchFiles(dir, pattern string) ([]string, error) {
	matches, err := glob.Glob(path.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("Invalid source glob %s", pattern)
	}
	// Filter out directories
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			results = append(results, match)
		}
	}
	sort.Strings(results)
	return results, nil
}
