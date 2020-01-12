package store

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

// NotFound indicates an object does not exist
type NotFound string

func (e NotFound) Error() string { return string(e) }

// Item in storage
type Item struct {
	Key          string
	ETag         string
	Version      string
	Size         int64
	LastModified time.Time
	Meta         map[string]string
}

// Exists returns true if it is a valid Item
func (item Item) Exists() bool {
	return item.Key != "" && item.ETag != "" && item.Size > 0
}

// Store is an interface to Get and Put items into storage
type Store interface {

	// Get an item from the Store
	Get(ctx context.Context, key, dst string) error

	// Put an item in the Store
	Put(ctx context.Context, key, src string, meta map[string]string) error

	// List contents of the Store
	List(ctx context.Context, prefix string) ([]string, error)

	// Head checks if the item exists in the store
	Head(ctx context.Context, key string) (Item, error)
}

type s3Store struct {
	bucket string
	api    s3iface.S3API
}

// NewS3 returns an S3 storage interface
func NewS3(api s3iface.S3API, bucket string) Store {
	return &s3Store{
		bucket: bucket,
		api:    api,
	}
}

// Get an item from storage
func (s *s3Store) Get(ctx context.Context, key, dst string) error {

	object, err := s.api.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return NotFound(fmt.Sprintf("Not found: %s/%s", s.bucket, key))
		}
		return fmt.Errorf("Failed to get %s/%s: %s", s.bucket, key, err)
	}

	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Failed to create file: %s", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, object.Body); err != nil {
		return fmt.Errorf("Failed to write file: %s", err)
	}
	return nil
}

// Put an item in the Store
func (s *s3Store) Put(ctx context.Context, key, src string, meta map[string]string) error {

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Failed to open file %s: %s", src, err)
	}
	defer f.Close()

	metadata := map[string]*string{}
	if meta != nil {
		for k, v := range meta {
			metadata[k] = aws.String(v)
		}
	}

	_, err = s.api.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		Body:     f,
		Metadata: metadata,
		ACL:      aws.String("bucket-owner-full-control"),
	})
	if err != nil {
		return fmt.Errorf("Failed to upload %s: %s", key, err)
	}
	return nil
}

// List contents of the Store
func (s *s3Store) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	err := s.api.ListObjectsPagesWithContext(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		for _, obj := range p.Contents {
			keys = append(keys, *obj.Key)
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to list store: %s", err)
	}
	return keys, nil
}

// Head checks if the item exists in the store
func (s *s3Store) Head(ctx context.Context, key string) (Item, error) {
	output, err := s.api.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFound(err) {
			return Item{Key: key}, NotFound(fmt.Sprintf("Not found: %s/%s",
				s.bucket, key))
		}
		return Item{Key: key}, fmt.Errorf("Head failed %s: %s", key, err)
	}
	item := Item{
		Key:  key,
		Meta: map[string]string{},
	}
	if output.VersionId != nil {
		item.Version = *output.VersionId
	}
	if output.ETag != nil {
		item.ETag = *output.ETag
	}
	if output.ContentLength != nil {
		item.Size = *output.ContentLength
	}
	if output.LastModified != nil {
		item.LastModified = *output.LastModified
	}
	if output.Metadata != nil {
		for k, v := range output.Metadata {
			item.Meta[k] = *v
		}
	}
	return item, nil
}

// Download files from the store with the given prefix to the given dst dir
func Download(ctx context.Context, store Store, prefix, dstDir string, ignore []string) ([]string, error) {
	var downloaded []string
	keys, err := store.List(ctx, prefix)
	if err != nil {
		return downloaded, err
	}
	// Could parallelize this
	for _, key := range keys {
		for _, ignorePrefix := range ignore {
			if strings.HasPrefix(key, ignorePrefix) {
				continue
			}
		}
		localPath := path.Join(dstDir, strings.TrimPrefix(key, prefix))
		localDir := path.Dir(localPath)
		if _, err := os.Stat(localDir); err != nil {
			if err := os.MkdirAll(localDir, 0755); err != nil {
				return downloaded, err
			}
		}
		itemInfo, _ := store.Head(ctx, key)
		if itemInfo.Exists() {
			localSize, localModTime := fileStat(localPath)
			if localSize > 0 && localSize == itemInfo.Size {
				if localModTime.Equal(itemInfo.LastModified) || localModTime.After(itemInfo.LastModified) {
					continue // File already exists locally
				}
			}
		}
		if err := store.Get(ctx, key, localPath); err != nil {
			return downloaded, err
		}
		// fmt.Println("Downloaded", key)
		downloaded = append(downloaded, localPath)
	}
	return downloaded, nil
}

// Upload a local directory to the store at the specified prefix
func Upload(ctx context.Context, store Store, prefix, srcDir string, ignore []string) ([]string, error) {
	var uploaded []string
	if stat, err := os.Stat(srcDir); err != nil || !stat.IsDir() {
		return uploaded, fmt.Errorf("No directory %s: %s", srcDir, err)
	}
	callback := func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			return nil
		}
		relativePath := strings.TrimPrefix(filePath, srcDir)
		for _, ignorePrefix := range ignore {
			if strings.HasPrefix(relativePath, ignorePrefix) {
				continue
			}
		}
		key := path.Join(prefix, relativePath)
		if itemInfo, _ := store.Head(ctx, key); itemInfo.Exists() {
			if localSize, _ := fileStat(filePath); localSize == itemInfo.Size {
				return nil // File already exists in store
			}
		}
		if err := store.Put(ctx, key, filePath, nil); err != nil {
			return fmt.Errorf("Failed to upload %s to %s: %s", filePath, key, err)
		}
		// fmt.Println("Uploaded", key)
		uploaded = append(uploaded, key)
		return nil
	}
	if err := filepath.Walk(srcDir, callback); err != nil {
		return uploaded, fmt.Errorf("Failed to upload %s: %s", srcDir, err)
	}
	return uploaded, nil
}

// Returns the size and last modified timestamp for the file at the given path
func fileStat(name string) (int64, time.Time) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, time.Time{}
	}
	return info.Size(), info.ModTime()
}

func isNotFound(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "NotFound":
			return true
		}
	}
	return false
}
