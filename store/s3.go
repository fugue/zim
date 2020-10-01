// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package store

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

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
