package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/LuminalHQ/zim/sign"
)

type httpStore struct {
	signingURL string
	authToken  string
}

// NewHTTP returns an HTTP storage interface
func NewHTTP(signingURL, authToken string) Store {
	return &httpStore{signingURL, authToken}
}

func (s *httpStore) request(ctx context.Context, url string, input *sign.Input, output interface{}) error {
	if s.authToken == "" {
		return fmt.Errorf("ZIM_TOKEN is not set")
	}
	js, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("Failed to marshal request: %s", err)
	}
	cli := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(js))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.authToken))
	if err != nil {
		return fmt.Errorf("Failed to build request: %s", err)
	}

	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("Request failed: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMessage, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Request failed (%d): %s", resp.StatusCode, errMessage)
	}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return fmt.Errorf("Failed to decode response: %s", err)
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
		return fmt.Errorf("Failed to build request: %s", err)
	}
	defer resp.Body.Close()

	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Failed to create file: %s", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("Failed to write file: %s", err)
	}
	return nil
}

// Put an item in the Store
func (s *httpStore) Put(ctx context.Context, key, src string, meta map[string]string) error {

	input := sign.Input{Method: "PUT", Name: key, Metadata: meta}
	output, err := s.requestSign(ctx, &input)
	if err != nil {
		return fmt.Errorf("Failed to sign PUT request %s: %s", src, err)
	}

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Failed to open file %s: %s", src, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("Failed to stat file %s: %s", src, err)
	}

	cli := &http.Client{}
	req, err := http.NewRequest("PUT", output.URL, f)
	req.ContentLength = stat.Size()
	for k, v := range meta {
		hdr := fmt.Sprintf("x-amz-meta-%s", strings.ToLower(k))
		req.Header.Add(hdr, v)
	}

	if err != nil {
		return fmt.Errorf("Failed to create request: %s", err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to make request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		message, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("PUT failed %s: %s", key, message)
	}
	return nil
}

// List contents of the Store
func (s *httpStore) List(ctx context.Context, prefix string) ([]string, error) {
	return []string{}, errors.New("Unimplemented")
}

// Head checks if the item exists in the store
func (s *httpStore) Head(ctx context.Context, key string) (Item, error) {
	input := sign.Input{Name: key}
	output, err := s.requestHead(ctx, &input)
	if err != nil {
		return Item{}, err
	}
	if output.ETag == "" {
		return Item{}, NotFound(fmt.Sprintf("Not found: %s", key))
	}
	return Item{
		Key:          output.Key,
		Version:      output.Version,
		ETag:         output.ETag,
		Size:         output.Size,
		LastModified: output.LastModified,
		Meta:         output.Metadata,
	}, nil
}
