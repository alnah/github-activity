package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFormatDescription(t *testing.T) {
	tests := []struct {
		name     string
		event    GitHubEvent
		expected string
	}{
		{
			name: "PushEvent with single commit",
			event: GitHubEvent{
				Type: "PushEvent",
				Repo: Repo{Name: "user/repo"},
				Payload: json.RawMessage(`{
					"size": 1,
					"commits": [{"sha": "abc123", "message": "test commit"}],
					"ref": "refs/heads/main"
				}`),
			},
			expected: "Pushed 1 commit to user/repo (branch: main)",
		},
		{
			name: "PushEvent with multiple commits",
			event: GitHubEvent{
				Type: "PushEvent",
				Repo: Repo{Name: "user/repo"},
				Payload: json.RawMessage(`{
					"size": 3,
					"commits": [
						{"sha": "abc123", "message": "commit 1"},
						{"sha": "def456", "message": "commit 2"},
						{"sha": "ghi789", "message": "commit 3"}
					],
					"ref": "refs/heads/develop"
				}`),
			},
			expected: "Pushed 3 commits to user/repo (branch: develop)",
		},
		{
			name: "CreateEvent for branch",
			event: GitHubEvent{
				Type: "CreateEvent",
				Repo: Repo{Name: "user/repo"},
				Payload: json.RawMessage(`{
					"ref_type": "branch",
					"ref": "feature/new-feature"
				}`),
			},
			expected: "Created branch 'feature/new-feature' in user/repo",
		},
		{
			name: "WatchEvent",
			event: GitHubEvent{
				Type:    "WatchEvent",
				Repo:    Repo{Name: "facebook/react"},
				Payload: json.RawMessage(`{}`),
			},
			expected: "Starred facebook/react",
		},
		{
			name: "IssuesEvent with issue details",
			event: GitHubEvent{
				Type: "IssuesEvent",
				Repo: Repo{Name: "user/repo"},
				Payload: json.RawMessage(`{
					"action": "opened",
					"issue": {
						"number": 42,
						"title": "Bug: Something is broken",
						"state": "open"
					}
				}`),
			},
			expected: "Opened issue #42 in user/repo: Bug: Something is broken",
		},
		{
			name: "ForkEvent with forkee",
			event: GitHubEvent{
				Type: "ForkEvent",
				Repo: Repo{Name: "original/repo"},
				Payload: json.RawMessage(`{
					"forkee": {
						"full_name": "user/forked-repo"
					}
				}`),
			},
			expected: "Forked original/repo to user/forked-repo",
		},
		{
			name: "Unknown event type",
			event: GitHubEvent{
				Type:    "UnknownEvent",
				Repo:    Repo{Name: "user/repo"},
				Payload: json.RawMessage(`{}`),
			},
			expected: "UnknownEvent in user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.event.FormatDescription()
			if result != tt.expected {
				t.Errorf("FormatDescription() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"opened", "Opened"},
		{"closed", "Closed"},
		{"UPPERCASE", "Uppercase"},
		{"", ""},
		{"a", "A"},
		{"ABC", "Abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := titleCase(tt.input)
			if result != tt.expected {
				t.Errorf("titleCase(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPushPayload_GetBranch(t *testing.T) {
	tests := []struct {
		name     string
		payload  PushPayload
		expected string
	}{
		{
			name:     "main branch",
			payload:  PushPayload{Ref: "refs/heads/main"},
			expected: "main",
		},
		{
			name:     "feature branch",
			payload:  PushPayload{Ref: "refs/heads/feature/new-feature"},
			expected: "feature/new-feature",
		},
		{
			name:     "branch without prefix",
			payload:  PushPayload{Ref: "develop"},
			expected: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payload.GetBranch()
			if result != tt.expected {
				t.Errorf("GetBranch() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommit_GetShortSHA(t *testing.T) {
	tests := []struct {
		name     string
		commit   Commit
		expected string
	}{
		{
			name:     "normal SHA",
			commit:   Commit{SHA: "abc123def456ghi789"},
			expected: "abc123d",
		},
		{
			name:     "short SHA",
			commit:   Commit{SHA: "abc"},
			expected: "abc",
		},
		{
			name:     "empty SHA",
			commit:   Commit{SHA: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.commit.GetShortSHA()
			if result != tt.expected {
				t.Errorf("GetShortSHA() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommit_GetFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		commit   Commit
		expected string
	}{
		{
			name:     "single line message",
			commit:   Commit{Message: "Fix bug in parser"},
			expected: "Fix bug in parser",
		},
		{
			name:     "multi-line message",
			commit:   Commit{Message: "Fix bug in parser\n\nThis fixes issue #123"},
			expected: "Fix bug in parser",
		},
		{
			name:     "empty message",
			commit:   Commit{Message: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.commit.GetFirstLine()
			if result != tt.expected {
				t.Errorf("GetFirstLine() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		maxLength int
		expected  string
	}{
		{
			name:      "short message",
			message:   "Short",
			maxLength: 10,
			expected:  "Short",
		},
		{
			name:      "exact length",
			message:   "Exact ten!",
			maxLength: 10,
			expected:  "Exact ten!",
		},
		{
			name:      "long message",
			message:   "This is a very long message that should be truncated",
			maxLength: 20,
			expected:  "This is a very lo...",
		},
		{
			name:      "very short max",
			message:   "Hello",
			maxLength: 3,
			expected:  "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateMessage(tt.message, tt.maxLength)
			if result != tt.expected {
				t.Errorf("TruncateMessage(%q, %d) = %v, want %v",
					tt.message, tt.maxLength, result, tt.expected)
			}
		})
	}
}

func TestEventFilter_Matches(t *testing.T) {
	tests := []struct {
		name     string
		filter   EventFilter
		event    GitHubEvent
		expected bool
	}{
		{
			name:     "no filter matches any",
			filter:   EventFilter{},
			event:    GitHubEvent{Type: "PushEvent"},
			expected: true,
		},
		{
			name:     "type filter matches",
			filter:   EventFilter{Type: "PushEvent"},
			event:    GitHubEvent{Type: "PushEvent"},
			expected: true,
		},
		{
			name:     "type filter case insensitive",
			filter:   EventFilter{Type: "pushevent"},
			event:    GitHubEvent{Type: "PushEvent"},
			expected: true,
		},
		{
			name:     "type filter no match",
			filter:   EventFilter{Type: "IssuesEvent"},
			event:    GitHubEvent{Type: "PushEvent"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(tt.event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetCommitDetails(t *testing.T) {
	t.Run("PushEvent returns commits", func(t *testing.T) {
		event := GitHubEvent{
			Type: "PushEvent",
			Payload: json.RawMessage(`{
				"commits": [
					{"sha": "abc123", "message": "commit 1"},
					{"sha": "def456", "message": "commit 2"}
				]
			}`),
		}

		commits, err := event.GetCommitDetails()
		if err != nil {
			t.Fatalf("GetCommitDetails() error = %v", err)
		}

		if len(commits) != 2 {
			t.Errorf("GetCommitDetails() returned %d commits, want 2", len(commits))
		}
	})

	t.Run("non-PushEvent returns error", func(t *testing.T) {
		event := GitHubEvent{
			Type:    "IssuesEvent",
			Payload: json.RawMessage(`{}`),
		}

		_, err := event.GetCommitDetails()
		if err == nil {
			t.Error("GetCommitDetails() should return error for non-PushEvent")
		}
	})
}

func TestGetAvailableEventTypes(t *testing.T) {
	types := GetAvailableEventTypes()

	// Check some expected types exist
	expectedTypes := []EventType{
		EventTypePush,
		EventTypeCreate,
		EventTypeIssues,
		EventTypePullRequest,
	}

	for _, expectedType := range expectedTypes {
		if _, exists := types[expectedType]; !exists {
			t.Errorf("GetAvailableEventTypes() missing %s", expectedType)
		}
	}

	// Verify all entries have descriptions
	for eventType, description := range types {
		if description == "" {
			t.Errorf("Event type %s has empty description", eventType)
		}
	}
}

func TestPayloadParsing(t *testing.T) {
	// Test PushPayload parsing
	pushJSON := `{
		"size": 2,
		"commits": [
			{"sha": "abc123", "message": "First commit", "author": {"name": "John", "email": "john@example.com"}},
			{"sha": "def456", "message": "Second commit", "author": {"name": "Jane", "email": "jane@example.com"}}
		],
		"ref": "refs/heads/main"
	}`

	var pushPayload PushPayload
	err := json.Unmarshal([]byte(pushJSON), &pushPayload)
	if err != nil {
		t.Fatalf("Failed to parse PushPayload: %v", err)
	}

	if pushPayload.Size != 2 {
		t.Errorf("PushPayload.Size = %v, want %v", pushPayload.Size, 2)
	}

	if len(pushPayload.Commits) != 2 {
		t.Errorf("PushPayload.Commits length = %v, want %v", len(pushPayload.Commits), 2)
	}

	if pushPayload.Ref != "refs/heads/main" {
		t.Errorf("PushPayload.Ref = %v, want %v", pushPayload.Ref, "refs/heads/main")
	}
}

// Benchmark for event formatting
func BenchmarkFormatDescription(b *testing.B) {
	event := GitHubEvent{
		Type: "PushEvent",
		Repo: Repo{Name: "user/repo"},
		Payload: json.RawMessage(`{
			"size": 3,
			"commits": [
				{"sha": "abc123", "message": "commit 1"},
				{"sha": "def456", "message": "commit 2"},
				{"sha": "ghi789", "message": "commit 3"}
			],
			"ref": "refs/heads/main"
		}`),
		CreatedAt: time.Now(),
	}

	for i := 0; i < b.N; i++ {
		_ = event.FormatDescription()
	}
}
