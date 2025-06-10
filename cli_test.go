package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// MockOutputFormatter for testing CLI output
type MockOutputFormatter struct {
	activities         []ActivitySummary
	detailedActivities []DetailedActivity
}

func (m *MockOutputFormatter) FormatActivities(w io.Writer, activities []ActivitySummary) {
	m.activities = activities
}

func (m *MockOutputFormatter) FormatDetailedActivities(w io.Writer, activities []DetailedActivity) {
	m.detailedActivities = activities
}

func TestCLI_parseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected CLIFlags
	}{
		{
			name: "default flags",
			args: []string{"github-activity", "testuser"},
			expected: CLIFlags{
				EventType: "",
				Limit:     30,
				Detailed:  false,
				ListTypes: false,
			},
		},
		{
			name: "with type filter",
			args: []string{"github-activity", "-type=PushEvent", "testuser"},
			expected: CLIFlags{
				EventType: "PushEvent",
				Limit:     30,
				Detailed:  false,
				ListTypes: false,
			},
		},
		{
			name: "with limit",
			args: []string{"github-activity", "-limit=5", "testuser"},
			expected: CLIFlags{
				EventType: "",
				Limit:     5,
				Detailed:  false,
				ListTypes: false,
			},
		},
		{
			name: "with detailed flag",
			args: []string{"github-activity", "-detailed", "testuser"},
			expected: CLIFlags{
				EventType: "",
				Limit:     30,
				Detailed:  true,
				ListTypes: false,
			},
		},
		{
			name: "list types flag",
			args: []string{"github-activity", "-list-types"},
			expected: CLIFlags{
				EventType: "",
				Limit:     30,
				Detailed:  false,
				ListTypes: true,
			},
		},
		{
			name: "all flags",
			args: []string{
				"github-activity",
				"-type=IssuesEvent",
				"-limit=10",
				"-detailed",
				"testuser",
			},
			expected: CLIFlags{
				EventType: "IssuesEvent",
				Limit:     10,
				Detailed:  true,
				ListTypes: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)

			cli := NewCLI(nil)
			flags := cli.parseFlags(tt.args)

			if flags.EventType != tt.expected.EventType {
				t.Errorf("EventType = %v, want %v", flags.EventType, tt.expected.EventType)
			}
			if flags.Limit != tt.expected.Limit {
				t.Errorf("Limit = %v, want %v", flags.Limit, tt.expected.Limit)
			}
			if flags.Detailed != tt.expected.Detailed {
				t.Errorf("Detailed = %v, want %v", flags.Detailed, tt.expected.Detailed)
			}
			if flags.ListTypes != tt.expected.ListTypes {
				t.Errorf("ListTypes = %v, want %v", flags.ListTypes, tt.expected.ListTypes)
			}
		})
	}
}

func TestCLI_Run(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		setupService func() *ActivityService
		expectedCode int
		checkOutput  func(t *testing.T, output string)
	}{
		{
			name: "no arguments shows usage",
			args: []string{"github-activity"},
			setupService: func() *ActivityService {
				return NewActivityService(NewMockEventRepository(nil, nil))
			},
			expectedCode: 1,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Usage:") {
					t.Error("Expected usage information in output")
				}
			},
		},
		{
			name: "list types",
			args: []string{"github-activity", "-list-types"},
			setupService: func() *ActivityService {
				return NewActivityService(NewMockEventRepository(nil, nil))
			},
			expectedCode: 0,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Available event types:") {
					t.Error("Expected event types list in output")
				}
			},
		},
		{
			name: "successful activity fetch",
			args: []string{"github-activity", "testuser"},
			setupService: func() *ActivityService {
				events := []GitHubEvent{
					{
						ID:        "1",
						Type:      "PushEvent",
						Repo:      Repo{Name: "user/repo"},
						CreatedAt: time.Now(),
						Payload:   json.RawMessage(`{"size": 1, "ref": "refs/heads/main"}`),
					},
				}
				repo := NewMockEventRepository(events, nil)
				return NewActivityService(repo)
			},
			expectedCode: 0,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Fetching GitHub activity for user: testuser") {
					t.Error("Expected fetch message in output")
				}
			},
		},
		{
			name: "error from service",
			args: []string{"github-activity", "nonexistent"},
			setupService: func() *ActivityService {
				repo := NewMockEventRepository(nil, &RepositoryError{
					Code:    "USER_NOT_FOUND",
					Message: "User not found",
				})
				return NewActivityService(repo)
			},
			expectedCode: 1,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Error:") {
					t.Error("Expected error message in output")
				}
			},
		},
		{
			name: "no activities found",
			args: []string{"github-activity", "emptyuser"},
			setupService: func() *ActivityService {
				// Return empty events array
				repo := NewMockEventRepository([]GitHubEvent{}, nil)
				return NewActivityService(repo)
			},
			expectedCode: 0,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "No recent activity found") {
					t.Error("Expected 'no activity' message in output")
				}
			},
		},
		{
			name: "with type filter no matches",
			args: []string{"github-activity", "-type=IssuesEvent", "testuser"},
			setupService: func() *ActivityService {
				events := []GitHubEvent{
					{
						ID:   "1",
						Type: "PushEvent", // Different type
						Repo: Repo{Name: "user/repo"},
					},
				}
				repo := NewMockEventRepository(events, nil)
				return NewActivityService(repo)
			},
			expectedCode: 0,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "No 'IssuesEvent' events found") {
					t.Error("Expected filtered no events message")
				}
			},
		},
		{
			name: "invalid event type",
			args: []string{"github-activity", "-type=InvalidEvent", "testuser"},
			setupService: func() *ActivityService {
				return NewActivityService(NewMockEventRepository(nil, nil))
			},
			expectedCode: 1,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Error:") &&
					!strings.Contains(output, "invalid event type") {
					t.Error("Expected invalid event type error")
				}
			},
		},
		{
			name: "negative limit",
			args: []string{"github-activity", "-limit=-5", "testuser"},
			setupService: func() *ActivityService {
				return NewActivityService(NewMockEventRepository(nil, nil))
			},
			expectedCode: 1,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Error:") &&
					!strings.Contains(output, "limit cannot be negative") {
					t.Error("Expected negative limit error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Create a buffer to capture output
			var outputBuf bytes.Buffer

			// Create a custom CLI with mocked output
			service := tt.setupService()
			cli := NewCLI(service)

			// Temporarily redirect fmt output for testing
			// Save original stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr

			// Create a pipe
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			// Run the CLI in a goroutine
			done := make(chan struct{})
			var code int
			go func() {
				code = cli.Run(tt.args)
				_ = w.Close()
				close(done)
			}()

			// Read from the pipe
			_, _ = io.Copy(&outputBuf, r)
			<-done

			// Restore stdout and stderr
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			output := outputBuf.String()

			// Check exit code
			if code != tt.expectedCode {
				t.Errorf("Exit code = %v, want %v\nOutput: %s", code, tt.expectedCode, output)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestConsoleOutputFormatter_FormatActivities(t *testing.T) {
	activities := []ActivitySummary{
		{Description: "Pushed 1 commit to user/repo"},
		{Description: "Opened issue #42 in user/repo"},
	}

	// Use a buffer instead of capturing stdout
	var buf bytes.Buffer
	formatter := &ConsoleOutputFormatter{}
	formatter.FormatActivities(&buf, activities)

	output := buf.String()

	// Check output contains activities
	for _, activity := range activities {
		if !strings.Contains(output, activity.Description) {
			t.Errorf("Output missing activity: %s", activity.Description)
		}
		if !strings.Contains(output, "- ") {
			t.Error("Output should have bullet points")
		}
	}
}

func TestConsoleOutputFormatter_FormatDetailedActivities(t *testing.T) {
	activities := []DetailedActivity{
		{
			ActivitySummary: ActivitySummary{
				Description: "Pushed 2 commits to user/repo",
				Type:        "PushEvent",
				Timestamp:   "2024-01-15 10:30:00",
			},
			Commits: []CommitSummary{
				{SHA: "abc123", Message: "First commit"},
				{SHA: "def456", Message: "Second commit"},
			},
		},
	}

	// Use a buffer instead of capturing stdout
	var buf bytes.Buffer
	formatter := &ConsoleOutputFormatter{}
	formatter.FormatDetailedActivities(&buf, activities)

	output := buf.String()

	// Check output contains expected elements
	expectedStrings := []string{
		"Pushed 2 commits",
		"Time: 2024-01-15 10:30:00",
		"Type: PushEvent",
		"Commits:",
		"abc123: First commit",
		"def456: Second commit",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Output missing: %s", expected)
		}
	}
}

func TestCLI_displayActivities(t *testing.T) {
	t.Run("no activities", func(t *testing.T) {
		service := NewActivityService(NewMockEventRepository([]GitHubEvent{}, nil))
		cli := NewCLI(service)
		cli.output = &MockOutputFormatter{}

		code := cli.displayActivities("testuser", EventFilter{})
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}
	})

	t.Run("with activities", func(t *testing.T) {
		events := []GitHubEvent{
			{ID: "1", Type: "PushEvent", Repo: Repo{Name: "user/repo"}},
		}
		service := NewActivityService(NewMockEventRepository(events, nil))

		mockOutput := &MockOutputFormatter{}
		cli := NewCLI(service)
		cli.output = mockOutput

		code := cli.displayActivities("testuser", EventFilter{})
		if code != 0 {
			t.Errorf("Expected exit code 0, got %d", code)
		}

		if len(mockOutput.activities) != 1 {
			t.Errorf("Expected 1 activity to be formatted, got %d", len(mockOutput.activities))
		}
	})
}

func TestCLI_displayDetailedActivities(t *testing.T) {
	events := []GitHubEvent{
		{
			ID:    "1",
			Type:  "PushEvent",
			Repo:  Repo{Name: "user/repo"},
			Actor: Actor{Login: "testuser"},
		},
	}
	service := NewActivityService(NewMockEventRepository(events, nil))

	mockOutput := &MockOutputFormatter{}
	cli := NewCLI(service)
	cli.output = mockOutput

	code := cli.displayDetailedActivities("testuser", EventFilter{})
	if code != 0 {
		t.Errorf("Expected exit code 0, got %d", code)
	}

	if len(mockOutput.detailedActivities) != 1 {
		t.Errorf("Expected 1 detailed activity to be formatted, got %d",
			len(mockOutput.detailedActivities))
	}
}
