package main

import (
	"testing"

	"github.com/fugue/zim/project"
)

// Swap out the paths to directories with many files for a more representative
// benchmark

func BenchmarkGlob(b *testing.B) {
	pat := "*.go"
	for i := 0; i < b.N; i++ {
		project.Glob(pat)
	}
}

func BenchmarkMatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		matchFiles(".", "*.go")
	}
}
