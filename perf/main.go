package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"time"

	glob "github.com/bmatcuk/doublestar"
	"github.com/fugue/zim/project"
)

func matchFiles(dir, pattern string) ([]string, error) {
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

func main() {
	var path, glob string
	var mode, show bool
	flag.StringVar(&path, "path", "", "Path to work with")
	flag.StringVar(&glob, "glob", "", "Glob pattern")
	flag.BoolVar(&mode, "old", false, "Use old glob")
	flag.BoolVar(&show, "show", false, "Show results")
	flag.Parse()

	if os.Getenv("ZIM_PROFILE") == "1" {
		f, err := os.Create("cpuprofile.out")
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	pattern := filepath.Join(path, glob)
	fmt.Println("Pattern:", pattern)

	start := time.Now()

	var results []string
	if !mode {
		fmt.Println("New glob")
		var err error
		results, err = project.Glob(pattern)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Old glob")
		var err error
		results, err = matchFiles(path, glob)
		if err != nil {
			log.Fatal(err)
		}
	}

	if show {
		for i, match := range results {
			fmt.Println(i, match)
		}
	}

	dt := time.Now().Sub(start)
	fmt.Printf("Found %d files in %v\n", len(results), dt)
}
