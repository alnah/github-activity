package main

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestActivityService_GetUserActivity(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		filter        EventFilter
		mockEvents    []GitHubEvent
		mockError     error
		expectedError bool
		expectedCount int
	}{
		{
			name:     "successful fetch with no filter",
			username: "testuser",
			filter:   EventFilter{},
			mockEvents: []GitHubEvent{
				{ID: "1", Type: "PushEvent", Repo: Repo{Name: "user/repo1"}},
				{ID: "2", Type: "IssuesEvent", Repo: Repo{Name: "user/repo2"}},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "with type filter",
			username: "testuser",
			filter:   EventFilter{Type: "PushEvent"},
			mockEvents: []GitHubEvent{
				{ID: "1", Type: "PushEvent", Repo: Repo{Name: "user/repo1"}},
				{ID: "2", Type: "IssuesEvent", Repo: Repo{Name: "user/repo2"}},
				{ID: "3", Type: "PushEvent", Repo: Repo{Name: "user/repo3"}},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "with limit",
			username: "testuser",
			filter:   EventFilter{MaxLimit: 1},
			mockEvents: []GitHubEvent{
				{ID: "1", Type: "PushEvent", Repo: Repo{Name: "user/repo1"}},
				{ID: "2", Type: "IssuesEvent", Repo: Repo{Name: "user/repo2"}},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:          "empty username",
			username:      "",
			filter:        EventFilter{},
			mockEvents:    nil,
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "repository error",
			username:      "testuser",
			filter:        EventFilter{},
			mockError:     errors.New("API error"),
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockEventRepository(tt.mockEvents, tt.mockError)
			service := NewActivityService(repo)

			activities, err := service.GetUserActivity(tt.username, tt.filter)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if len(activities) != tt.expectedCount {
					t.Errorf("Got %d activities, want %d", len(activities), tt.expectedCount)
				}
			}
		})
	}
}

func TestActivityService_GetUserActivityDetailed(t *testing.T) {
	pushEvent := GitHubEvent{
		ID:        "1",
		Type:      "PushEvent",
		Repo:      Repo{Name: "user/repo"},
		Actor:     Actor{Login: "testuser"},
		CreatedAt: time.Now(),
		Payload: json.RawMessage(`{
			"size": 2,
			"commits": [
				{"sha": "abc123", "message": "First commit", "author": {"name": "John"}},
				{"sha": "def456", "message": "Second commit with a very long message that should be truncated", "author": {"name": "Jane"}}
			],
			"ref": "refs/heads/main"
		}`),
	}

	repo := NewMockEventRepository([]GitHubEvent{pushEvent}, nil)
	service := NewActivityService(repo)

	activities, err := service.GetUserActivityDetailed("testuser", EventFilter{})
	if err != nil {
		t.Fatalf("GetUserActivityDetailed() error = %v", err)
	}

	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	activity := activities[0]

	// Check basic fields
	if activity.EventID != "1" {
		t.Errorf("EventID = %v, want %v", activity.EventID, "1")
	}

	if activity.ActorLogin != "testuser" {
		t.Errorf("ActorLogin = %v, want %v", activity.ActorLogin, "testuser")
	}

	// Check commit details
	if activity.CommitCount != 2 {
		t.Errorf("CommitCount = %v, want %v", activity.CommitCount, 2)
	}

	if len(activity.Commits) != 2 {
		t.Errorf("Commits length = %v, want %v", len(activity.Commits), 2)
	}

	// Check first commit
	if activity.Commits[0].SHA != "abc123" {
		t.Errorf("First commit SHA = %v, want %v", activity.Commits[0].SHA, "abc123")
	}

	// Check message truncation
	if len(activity.Commits[1].Message) > 60 {
		t.Errorf(
			"Commit message should be truncated to 60 chars, got %d",
			len(activity.Commits[1].Message),
		)
	}
}

func TestActivityService_GetEventTypeStatistics(t *testing.T) {
	mockEvents := []GitHubEvent{
		{Type: "PushEvent"},
		{Type: "PushEvent"},
		{Type: "IssuesEvent"},
		{Type: "PushEvent"},
		{Type: "WatchEvent"},
		{Type: "IssuesEvent"},
	}

	repo := NewMockEventRepository(mockEvents, nil)
	service := NewActivityService(repo)

	stats, err := service.GetEventTypeStatistics("testuser")
	if err != nil {
		t.Fatalf("GetEventTypeStatistics() error = %v", err)
	}

	expected := map[string]int{
		"PushEvent":   3,
		"IssuesEvent": 2,
		"WatchEvent":  1,
	}

	for eventType, count := range expected {
		if stats[eventType] != count {
			t.Errorf("Stats[%s] = %d, want %d", eventType, stats[eventType], count)
		}
	}
}

func TestActivityService_GetRecentRepositories(t *testing.T) {
	mockEvents := []GitHubEvent{
		{Repo: Repo{Name: "user/repo1"}},
		{Repo: Repo{Name: "user/repo2"}},
		{Repo: Repo{Name: "user/repo1"}}, // Duplicate
		{Repo: Repo{Name: "user/repo3"}},
		{Repo: Repo{Name: "user/repo2"}}, // Duplicate
		{Repo: Repo{Name: "user/repo4"}},
	}

	repo := NewMockEventRepository(mockEvents, nil)
	service := NewActivityService(repo)

	t.Run("no limit", func(t *testing.T) {
		repos, err := service.GetRecentRepositories("testuser", 0)
		if err != nil {
			t.Fatalf("GetRecentRepositories() error = %v", err)
		}

		expected := []string{"user/repo1", "user/repo2", "user/repo3", "user/repo4"}
		if len(repos) != len(expected) {
			t.Errorf("Got %d repos, want %d", len(repos), len(expected))
		}

		for i, repo := range repos {
			if repo != expected[i] {
				t.Errorf("Repo[%d] = %v, want %v", i, repo, expected[i])
			}
		}
	})

	t.Run("with limit", func(t *testing.T) {
		repos, err := service.GetRecentRepositories("testuser", 2)
		if err != nil {
			t.Fatalf("GetRecentRepositories() error = %v", err)
		}

		if len(repos) != 2 {
			t.Errorf("Got %d repos, want 2", len(repos))
		}

		expected := []string{"user/repo1", "user/repo2"}
		for i, repo := range repos {
			if repo != expected[i] {
				t.Errorf("Repo[%d] = %v, want %v", i, repo, expected[i])
			}
		}
	})
}

func TestActivityOptions_Validate(t *testing.T) {
	tests := []struct {
		name        string
		options     ActivityOptions
		expectError bool
	}{
		{
			name:        "valid options",
			options:     ActivityOptions{EventType: "PushEvent", Limit: 10},
			expectError: false,
		},
		{
			name:        "empty event type is valid",
			options:     ActivityOptions{EventType: "", Limit: 10},
			expectError: false,
		},
		{
			name:        "negative limit",
			options:     ActivityOptions{Limit: -1},
			expectError: true,
		},
		{
			name:        "invalid event type",
			options:     ActivityOptions{EventType: "InvalidEvent"},
			expectError: true,
		},
		{
			name:        "case insensitive event type",
			options:     ActivityOptions{EventType: "pushevent"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDefaultActivityOptions(t *testing.T) {
	options := DefaultActivityOptions()

	if options.EventType != "" {
		t.Errorf("Default EventType = %v, want empty", options.EventType)
	}

	if options.Limit != 30 {
		t.Errorf("Default Limit = %v, want 30", options.Limit)
	}

	if options.ShowDetailed {
		t.Error("Default ShowDetailed should be false")
	}
}

func TestActivityService_createActivitySummary(t *testing.T) {
	event := GitHubEvent{
		ID:        "123",
		Type:      "PushEvent",
		Repo:      Repo{Name: "user/repo"},
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Payload: json.RawMessage(`{
			"size": 1,
			"ref": "refs/heads/main"
		}`),
	}

	service := NewActivityService(nil)
	summary := service.createActivitySummary(event)

	if summary.Type != "PushEvent" {
		t.Errorf("Type = %v, want PushEvent", summary.Type)
	}

	if summary.Repository != "user/repo" {
		t.Errorf("Repository = %v, want user/repo", summary.Repository)
	}

	expectedTime := "2024-01-15 10:30:00"
	if summary.Timestamp != expectedTime {
		t.Errorf("Timestamp = %v, want %v", summary.Timestamp, expectedTime)
	}

	if summary.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestActivityService_ErrorHandling(t *testing.T) {
	t.Run("whitespace username", func(t *testing.T) {
		service := NewActivityService(nil)
		_, err := service.GetUserActivity("   ", EventFilter{})
		if err == nil {
			t.Error("Expected error for whitespace username")
		}
	})

	t.Run("repository error propagation", func(t *testing.T) {
		expectedErr := errors.New("API error")
		repo := NewMockEventRepository(nil, expectedErr)
		service := NewActivityService(repo)

		_, err := service.GetUserActivity("testuser", EventFilter{})
		if err == nil {
			t.Error("Expected error to be propagated")
		}
	})
}
