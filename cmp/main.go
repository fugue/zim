package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LuminalHQ/zim/zip"
)

func tmpDir() string {
	dir, err := ioutil.TempDir("", "zim-")
	if err != nil {
		panic(err)
	}
	return dir
}

func readDir(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	return files
}

func unzip(src, dst string) {
	if err := zip.Unzip(src, dst); err != nil {
		panic(err)
	}
}

func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// FileSetOpts contains options used to create a FileSet
type FileSetOpts struct {
	Directory         string
	ExcludeExtensions []string
	ExcludeNames      []string
}

// File on disk
type File struct {
	Name string
	Extn string
	Path string
	Hash string
}

// FileSet tracks a set of files and their hashes
type FileSet struct {
	files map[string]*File
}

// Count of files in the set
func (fs *FileSet) Count() int {
	return len(fs.files)
}

// Paths returns a slice of all paths within the set
func (fs *FileSet) Paths() (result []string) {
	for path := range fs.files {
		result = append(result, path)
	}
	sort.Strings(result)
	return
}

// NewFileSet creates a FileSet with all files in the given dir
func NewFileSet(opts FileSetOpts) *FileSet {

	excludeExtensions := map[string]bool{}
	excludeNames := map[string]bool{}

	for _, value := range opts.ExcludeExtensions {
		excludeExtensions[value] = true
	}
	for _, value := range opts.ExcludeNames {
		excludeNames[value] = true
	}

	files := map[string]*File{}

	callback := func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			return nil
		}
		name := fileInfo.Name()
		extn := filepath.Ext(name)
		if excludeNames[name] || excludeExtensions[extn] {
			return nil
		}
		relPath, err := filepath.Rel(opts.Directory, filePath)
		if err != nil {
			panic(err)
		}
		files[relPath] = &File{
			Name: name,
			Extn: extn,
			Path: relPath,
			Hash: fileHash(filePath),
		}
		return nil
	}
	if err := filepath.Walk(opts.Directory, callback); err != nil {
		panic(err)
	}
	return &FileSet{files: files}
}

type unzipJob struct {
	src string
	dst string
}

func worker(jobs <-chan unzipJob, errs chan error) {
	for j := range jobs {
		unzip(j.src, j.dst)
	}
	errs <- nil
}

func unzipAll(srcDir string) string {

	dstDir := tmpDir()
	numWorkers := 6
	jobs := make(chan unzipJob, numWorkers)
	errs := make(chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(jobs, errs)
	}

	for _, f := range readDir(srcDir) {
		filePath := path.Join(srcDir, f.Name())
		dstPath := path.Join(dstDir, strings.Replace(f.Name(), ".zip", "", 1))
		if strings.HasSuffix(f.Name(), ".zip") {
			jobs <- unzipJob{src: filePath, dst: dstPath}
		}
	}
	close(jobs)

	for i := 0; i < numWorkers; i++ {
		<-errs
	}
	return dstDir
}

// Diff between FileSets
type Diff struct {
	MissingA  []string
	MissingB  []string
	Different []string
	Same      []string
}

// IsNil returns true if there is no difference between the two file sets
func (d *Diff) IsNil() bool {
	return len(d.MissingA) == 0 && len(d.MissingB) == 0 && len(d.Different) == 0
}

// Size returns the total number of items compared
func (d *Diff) Size() int {
	return len(d.MissingA) + len(d.MissingB) + len(d.Different) + len(d.Same)
}

// NewDiff returns a difference between
func NewDiff(a, b *FileSet) *Diff {

	var missingA, missingB, different, same []string

	done := map[string]bool{}

	for fpath, f := range a.files {
		other, found := b.files[fpath]
		if !found {
			missingB = append(missingB, fpath)
		} else {
			if f.Hash == other.Hash {
				same = append(same, fpath)
			} else {
				different = append(different, fpath)
			}
		}
		done[fpath] = true
	}

	for fpath := range b.files {
		if done[fpath] {
			continue
		}
		missingA = append(missingA, fpath)
	}

	sort.Strings(missingA)
	sort.Strings(missingB)
	sort.Strings(different)
	sort.Strings(same)

	return &Diff{
		MissingA:  missingA,
		MissingB:  missingB,
		Different: different,
		Same:      same,
	}
}

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Expected two directories as args")
		os.Exit(1)
	}

	dir1 := unzipAll(os.Args[1])
	dir2 := unzipAll(os.Args[2])

	fmt.Println(dir1)
	fmt.Println(dir2)

	extensions := []string{".pyc", ".so", ".dist-info", ".egg-info"}
	names := []string{"METADATA", "RECORD"}

	set1 := NewFileSet(FileSetOpts{
		Directory:         dir1,
		ExcludeNames:      names,
		ExcludeExtensions: extensions,
	})
	fmt.Println("Set1 count", set1.Count())

	set2 := NewFileSet(FileSetOpts{
		Directory:         dir2,
		ExcludeNames:      names,
		ExcludeExtensions: extensions,
	})
	fmt.Println("Set2 count", set2.Count())

	diff := NewDiff(set1, set2)

	fmt.Println("Same count", len(diff.Same))
	fmt.Println("Different count", len(diff.Different))
	fmt.Println("Missing in A", len(diff.MissingA))
	fmt.Println("Missing in B", len(diff.MissingB))

	for _, x := range diff.Different {
		fmt.Println("Different", x)
	}
	for _, x := range diff.MissingA {
		fmt.Println("MissingA", x)
	}
	for _, x := range diff.MissingB {
		fmt.Println("MissingB", x)
	}
}
