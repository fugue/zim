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

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/fugue/zim/sign"
	"github.com/fugue/zim/store"
	"github.com/hashicorp/go-retryablehttp"
)

type httpStore struct {
	signingURL string
	authToken  string
	client     *retryablehttp.Client
}

// New returns an HTTP storage interface
func New(signingURL, authToken string) store.Store {
	client := retryablehttp.NewClient()
	client.RetryMax = 4
	client.Logger = nil
	return &httpStore{
		signingURL: signingURL,
		authToken:  authToken,
		client:     client,
	}
}

func (s *httpStore) request(ctx context.Context, url string, input *sign.Input, output interface{}) error {
	if s.authToken == "" {
		return fmt.Errorf("ZIM_TOKEN is not set")
	}
	js, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %s", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(js))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.authToken))
	if err != nil {
		return fmt.Errorf("failed to build request: %s", err)
	}
	resp, err := s.client.StandardClient().Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMessage, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, errMessage)
	}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return fmt.Errorf("failed to decode response: %s", err)
	}
	return nil
}

func (s *httpStore) requestSign(ctx context.Context, input *sign.Input) (*sign.Output, error) {
	u, err := url.Parse(s.signingURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "sign")
	var output *sign.Output
	if err := s.request(ctx, u.String(), input, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func (s *httpStore) requestHead(ctx context.Context, input *sign.Input) (*sign.Item, error) {
	u, err := url.Parse(s.signingURL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "head")
	var output *sign.Item
	if err := s.request(ctx, u.String(), input, &output); err != nil {
		return nil, err
	}
	return output, nil
}

// Get an item from storage
func (s *httpStore) Get(ctx context.Context, key, dst string) error {

	input := sign.Input{Method: "GET", Name: key}
	output, err := s.requestSign(ctx, &input)
	if err != nil {
		return err
	}

	resp, err := http.Get(output.URL)
	if err != nil {
		return fmt.Errorf("failed to build request: %s", err)
	}
	defer resp.Body.Close()

	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create file: %s", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %s", err)
	}
	return nil
}

// Put an item in the Store
func (s *httpStore) Put(ctx context.Context, key, src string, meta map[string]string) error {

	input := sign.Input{Method: "PUT", Name: key, Metadata: meta}
	output, err := s.requestSign(ctx, &input)
	if err != nil {
		return fmt.Errorf("failed to sign PUT request %s: %s", src, err)
	}

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %s", src, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %s", src, err)
	}

	cli := &http.Client{}
	req, err := http.NewRequest("PUT", output.URL, f)
	req.ContentLength = stat.Size()
	for k, v := range meta {
		hdr := fmt.Sprintf("x-amz-meta-%s", strings.ToLower(k))
		req.Header.Add(hdr, v)
	}

	if err != nil {
		return fmt.Errorf("failed to create request: %s", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("PUT failed %s: %s", key, message)
	}
	return nil
}

// Head checks if the item exists in the store
func (s *httpStore) Head(ctx context.Context, key string) (store.ItemMeta, error) {
	input := sign.Input{Name: key}
	output, err := s.requestHead(ctx, &input)
	if err != nil {
		return store.ItemMeta{}, err
	}
	if output.ETag == "" {
		return store.ItemMeta{}, store.NotFound(fmt.Sprintf("Not found: %s", key))
	}
	return store.ItemMeta{Meta: output.Metadata}, nil
}
