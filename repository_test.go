package main

import (
	"errors"
	"testing"
	"time"
)

func TestEventCache_IsValid(t *testing.T) {
	cache := &EventCache{
		ttl: 5 * time.Minute,
	}

	t.Run("empty cache is invalid", func(t *testing.T) {
		if cache.IsValid("testuser") {
			t.Error("Empty cache should be invalid")
		}
	})

	t.Run("cache with different username is invalid", func(t *testing.T) {
		cache.Update("user1", []GitHubEvent{})
		if cache.IsValid("user2") {
			t.Error("Cache should be invalid for different username")
		}
	})

	t.Run("fresh cache is valid", func(t *testing.T) {
		cache.Update("testuser", []GitHubEvent{})
		if !cache.IsValid("testuser") {
			t.Error("Fresh cache should be valid")
		}
	})

	t.Run("expired cache is invalid", func(t *testing.T) {
		cache.Update("testuser", []GitHubEvent{})
		// Manually set timestamp to past
		cache.timestamp = time.Now().Add(-6 * time.Minute)
		if cache.IsValid("testuser") {
			t.Error("Expired cache should be invalid")
		}
	})
}

func TestEventCache_Update(t *testing.T) {
	cache := &EventCache{
		ttl: 5 * time.Minute,
	}

	events := []GitHubEvent{
		{ID: "1", Type: "PushEvent"},
		{ID: "2", Type: "WatchEvent"},
	}

	cache.Update("testuser", events)

	if cache.username != "testuser" {
		t.Errorf("Cache username = %v, want %v", cache.username, "testuser")
	}

	if len(cache.data) != 2 {
		t.Errorf("Cache data length = %v, want %v", len(cache.data), 2)
	}

	if cache.timestamp.IsZero() {
		t.Error("Cache timestamp should be set")
	}
}

func TestEventCache_Clear(t *testing.T) {
	cache := &EventCache{
		username:  "testuser",
		data:      []GitHubEvent{{ID: "1"}},
		timestamp: time.Now(),
		ttl:       5 * time.Minute,
	}

	cache.Clear()

	if cache.username != "" {
		t.Error("Cache username should be empty after clear")
	}

	if cache.data != nil {
		t.Error("Cache data should be nil after clear")
	}

	if !cache.timestamp.IsZero() {
		t.Error("Cache timestamp should be zero after clear")
	}
}

func TestEventCache_SetTTL(t *testing.T) {
	cache := &EventCache{}

	newTTL := 10 * time.Minute
	cache.SetTTL(newTTL)

	if cache.ttl != newTTL {
		t.Errorf("Cache TTL = %v, want %v", cache.ttl, newTTL)
	}
}

func TestMockEventRepository(t *testing.T) {
	t.Run("returns events when no error", func(t *testing.T) {
		expectedEvents := []GitHubEvent{
			{ID: "1", Type: "PushEvent"},
			{ID: "2", Type: "IssuesEvent"},
		}

		repo := NewMockEventRepository(expectedEvents, nil)
		events, err := repo.FetchEvents("testuser")
		if err != nil {
			t.Fatalf("FetchEvents() error = %v", err)
		}

		if len(events) != len(expectedEvents) {
			t.Errorf("FetchEvents() returned %d events, want %d", len(events), len(expectedEvents))
		}
	})

	t.Run("returns error when configured", func(t *testing.T) {
		expectedErr := errors.New("test error")
		repo := NewMockEventRepository(nil, expectedErr)

		_, err := repo.FetchEvents("testuser")

		if err != expectedErr {
			t.Errorf("FetchEvents() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestRepositoryError(t *testing.T) {
	t.Run("error without wrapped error", func(t *testing.T) {
		err := &RepositoryError{
			Code:    "TEST_ERROR",
			Message: "Test error message",
		}

		expected := "TEST_ERROR: Test error message"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}

		if err.Unwrap() != nil {
			t.Error("Unwrap() should return nil when no wrapped error")
		}
	})

	t.Run("error with wrapped error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped error")
		err := &RepositoryError{
			Code:    "TEST_ERROR",
			Message: "Test error message",
			Err:     wrappedErr,
		}

		expected := "TEST_ERROR: Test error message (wrapped error)"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}

		if err.Unwrap() != wrappedErr {
			t.Error("Unwrap() should return the wrapped error")
		}
	})
}

func TestNewGitHubAPIRepository(t *testing.T) {
	repo := NewGitHubAPIRepository()

	if repo.client == nil {
		t.Error("Repository client should not be nil")
	}

	if repo.client.Timeout != 10*time.Second {
		t.Errorf("Client timeout = %v, want %v", repo.client.Timeout, 10*time.Second)
	}

	if repo.cache == nil {
		t.Error("Repository cache should not be nil")
	}

	if repo.cache.ttl != 5*time.Minute {
		t.Errorf("Cache TTL = %v, want %v", repo.cache.ttl, 5*time.Minute)
	}

	if repo.userAgent != "github-activity-cli" {
		t.Errorf("User agent = %v, want %v", repo.userAgent, "github-activity-cli")
	}
}

// Integration test for caching behavior
func TestGitHubAPIRepository_Caching(t *testing.T) {
	// This would require mocking the HTTP client for a real test
	// For now, we'll test the caching logic structure

	repo := NewGitHubAPIRepository()

	// Simulate cache update
	testEvents := []GitHubEvent{
		{ID: "1", Type: "PushEvent"},
	}
	repo.cache.Update("testuser", testEvents)

	// Verify cache is used when valid
	if !repo.cache.IsValid("testuser") {
		t.Error("Cache should be valid immediately after update")
	}

	// Verify cache data
	if len(repo.cache.data) != 1 {
		t.Errorf("Cache should contain 1 event, got %d", len(repo.cache.data))
	}
}
