// Copyright 2026 Zenauth Ltd.

package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"go.uber.org/multierr"
	"golang.org/x/sync/semaphore"
)

const maxConcurrencyPerHost = 8

type Client struct {
	http       *http.Client
	semaphores map[string]*semaphore.Weighted
	auth       map[string]string
	mutex      sync.RWMutex
}

func NewClient() *Client {
	client := &Client{
		http:       http.DefaultClient,
		semaphores: make(map[string]*semaphore.Weighted),
		auth:       make(map[string]string),
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client.auth["github.com"] = "Bearer " + token
	}

	return client
}

func (c *Client) Get(ctx context.Context, url string) (io.ReadCloser, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	host := request.URL.Hostname()

	if auth := c.auth[host]; auth != "" {
		request.Header.Set("Authorization", auth)
	}

	if err := c.acquire(ctx, host); err != nil {
		return nil, err
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, multierr.Append(fmt.Errorf("GET %s: HTTP %d", url, response.StatusCode), response.Body.Close())
	}

	return responseBody{
		ReadCloser: response.Body,
		release: func() {
			c.release(host)
		},
	}, nil
}

func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, error) {
	responseBody, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer multierr.AppendInvoke(&err, multierr.Close(responseBody))

	return io.ReadAll(responseBody)
}

type responseBody struct {
	io.ReadCloser
	release func()
}

func (b responseBody) Close() error {
	b.release()
	return b.ReadCloser.Close()
}

func (c *Client) acquire(ctx context.Context, host string) error {
	c.mutex.RLock()
	semaphore, ok := c.semaphores[host]
	c.mutex.RUnlock()
	if !ok {
		c.mutex.Lock()
		semaphore, ok = c.semaphores[host]
		if !ok {
			semaphore = newSemaphore()
			c.semaphores[host] = semaphore
		}
		c.mutex.Unlock()
	}

	return semaphore.Acquire(ctx, 1)
}

func (c *Client) release(host string) {
	c.mutex.RLock()
	c.semaphores[host].Release(1)
	c.mutex.RUnlock()
}

func newSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(maxConcurrencyPerHost)
}
