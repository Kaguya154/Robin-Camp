package boxoffice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	endpointPath       = "/boxoffice"
	defaultHTTPTimeout = 5 * time.Second
	maxErrorBodyBytes  = 64 * 1024
)

var (
	// ErrMissingBaseURL indicates the upstream base URL was not provided.
	ErrMissingBaseURL = errors.New("boxoffice: base URL is required")
	// ErrMissingAPIKey indicates the upstream API key was not provided.
	ErrMissingAPIKey = errors.New("boxoffice: API key is required")
	// ErrEmptyTitle indicates the caller did not pass a movie title.
	ErrEmptyTitle = errors.New("boxoffice: title must not be empty")
	// ErrNotFound represents a 404 response from the upstream.
	ErrNotFound = errors.New("boxoffice: record not found")
)

// Client invokes the upstream Box Office API.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

// Option allows customizing the client.
type Option func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// NewClient builds a Client using the provided base URL and API key.
func NewClient(baseURL, apiKey string, opts ...Option) (*Client, error) {
	trimmedURL := strings.TrimSpace(baseURL)
	if trimmedURL == "" {
		return nil, ErrMissingBaseURL
	}

	trimmedKey := strings.TrimSpace(apiKey)
	if trimmedKey == "" {
		return nil, ErrMissingAPIKey
	}

	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		return nil, fmt.Errorf("boxoffice: parse base URL: %w", err)
	}

	client := &Client{
		baseURL: parsedURL,
		apiKey:  trimmedKey,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// NewFromEnv constructs a Client using BOXOFFICE_URL and BOXOFFICE_API_KEY environment variables.
func NewFromEnv(opts ...Option) (*Client, error) {
	return NewClient(os.Getenv("BOXOFFICE_URL"), os.Getenv("BOXOFFICE_API_KEY"), opts...)
}

// UpstreamError captures non-200 responses from the Box Office API.
type UpstreamError struct {
	StatusCode int
	Payload    *Error
}

func (e *UpstreamError) Error() string {
	if e == nil {
		return ""
	}
	if e.Payload != nil && e.Payload.Message != "" {
		return fmt.Sprintf("boxoffice: upstream %d: %s", e.StatusCode, e.Payload.Message)
	}
	return fmt.Sprintf("boxoffice: upstream %d", e.StatusCode)
}

func (e *UpstreamError) Is(target error) bool {
	if target == ErrNotFound && e.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

// GetMovieBoxOffice fetches box office information for a movie title.
func (c *Client) GetMovieBoxOffice(ctx context.Context, title string) (*BoxOffice, error) {
	if c == nil {
		return nil, errors.New("boxoffice: client is nil")
	}
	if ctx == nil {
		return nil, errors.New("boxoffice: context is nil")
	}
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return nil, ErrEmptyTitle
	}

	values := url.Values{}
	values.Set("title", trimmedTitle)

	reqURL := c.baseURL.ResolveReference(&url.URL{Path: endpointPath, RawQuery: values.Encode()})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("boxoffice: build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("boxoffice: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		upstreamErr := &UpstreamError{StatusCode: resp.StatusCode}
		upstreamErr.Payload = decodeError(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, upstreamErr.Error())
		}
		return nil, upstreamErr
	}

	var record BoxOffice
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("boxoffice: decode success payload: %w", err)
	}

	return &record, nil
}

func decodeError(r io.Reader) *Error {
	limited := io.LimitReader(r, maxErrorBodyBytes)
	var payload Error
	if err := json.NewDecoder(limited).Decode(&payload); err != nil {
		return nil
	}
	return &payload
}

// nullClient exists only to guard against nil receiver misuse; keep unexported to avoid API surface.
func nullClient() *Client { return nil }
