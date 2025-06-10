package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Repository Layer - Handles data fetching and caching

// EventRepository interface for fetching GitHub events
type EventRepository interface {
	FetchEvents(username string) ([]GitHubEvent, error)
}

// GitHubAPIRepository implements EventRepository using GitHub API
type GitHubAPIRepository struct {
	client    *http.Client
	cache     *EventCache
	userAgent string
}

// EventCache stores fetched events with TTL
type EventCache struct {
	data      []GitHubEvent
	username  string
	timestamp time.Time
	ttl       time.Duration
}

// NewGitHubAPIRepository creates a new repository instance
func NewGitHubAPIRepository() *GitHubAPIRepository {
	return &GitHubAPIRepository{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: &EventCache{
			ttl: 5 * time.Minute,
		},
		userAgent: "github-activity-cli",
	}
}

// FetchEvents fetches events for a given username with caching
func (r *GitHubAPIRepository) FetchEvents(username string) ([]GitHubEvent, error) {
	// Check cache first
	if r.cache.IsValid(username) {
		return r.cache.data, nil
	}

	// Fetch from API
	events, err := r.fetchFromAPI(username)
	if err != nil {
		return nil, err
	}

	// Update cache
	r.cache.Update(username, events)

	return events, nil
}

// fetchFromAPI performs the actual API call
func (r *GitHubAPIRepository) fetchFromAPI(username string) ([]GitHubEvent, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/events", username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle common HTTP errors
	switch resp.StatusCode {
	case 404:
		return nil, fmt.Errorf("user '%s' not found", username)
	case 401:
		return nil, fmt.Errorf("authentication required")
	case 403:
		return nil, fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var events []GitHubEvent
	if err := json.Unmarshal(body, &events); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return events, nil
}

// IsValid checks if the cache is valid for the given username
func (c *EventCache) IsValid(username string) bool {
	if c.username != username {
		return false
	}
	return time.Since(c.timestamp) < c.ttl
}

// Update updates the cache with new data
func (c *EventCache) Update(username string, events []GitHubEvent) {
	c.username = username
	c.data = events
	c.timestamp = time.Now()
}

// Clear clears the cache
func (c *EventCache) Clear() {
	c.username = ""
	c.data = nil
	c.timestamp = time.Time{}
}

// SetTTL sets the cache TTL duration
func (c *EventCache) SetTTL(ttl time.Duration) {
	c.ttl = ttl
}

// MockEventRepository is a mock implementation for testing
type MockEventRepository struct {
	events []GitHubEvent
	err    error
}

// NewMockEventRepository creates a new mock repository
func NewMockEventRepository(events []GitHubEvent, err error) *MockEventRepository {
	return &MockEventRepository{
		events: events,
		err:    err,
	}
}

// FetchEvents returns the mocked events or error
func (m *MockEventRepository) FetchEvents(username string) ([]GitHubEvent, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

// RepositoryError represents repository-specific errors
type RepositoryError struct {
	Code    string
	Message string
	Err     error
}

func (e *RepositoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// Common repository errors
var (
	ErrUserNotFound = &RepositoryError{
		Code:    "USER_NOT_FOUND",
		Message: "User not found",
	}
	ErrRateLimitExceeded = &RepositoryError{
		Code:    "RATE_LIMIT",
		Message: "API rate limit exceeded",
	}
	ErrNetworkError = &RepositoryError{
		Code:    "NETWORK_ERROR",
		Message: "Network error occurred",
	}
)
