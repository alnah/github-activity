package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Domain Models and Business Logic

// EventType represents the type of GitHub event
type EventType string

const (
	EventTypePush         EventType = "PushEvent"
	EventTypeCreate       EventType = "CreateEvent"
	EventTypeDelete       EventType = "DeleteEvent"
	EventTypeIssues       EventType = "IssuesEvent"
	EventTypePullRequest  EventType = "PullRequestEvent"
	EventTypeWatch        EventType = "WatchEvent"
	EventTypeFork         EventType = "ForkEvent"
	EventTypeIssueComment EventType = "IssueCommentEvent"
	EventTypePublic       EventType = "PublicEvent"
	EventTypeMember       EventType = "MemberEvent"
	EventTypeRelease      EventType = "ReleaseEvent"
)

// GitHubEvent represents a GitHub event from the API
type GitHubEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Actor     Actor           `json:"actor"`
	Repo      Repo            `json:"repo"`
	Payload   json.RawMessage `json:"payload"`
	Public    bool            `json:"public"`
	CreatedAt time.Time       `json:"created_at"`
}

// Actor represents the user who performed the action
type Actor struct {
	ID           int    `json:"id"`
	Login        string `json:"login"`
	DisplayLogin string `json:"display_login"`
	GravatarID   string `json:"gravatar_id"`
	URL          string `json:"url"`
	AvatarURL    string `json:"avatar_url"`
}

// Repo represents the repository
type Repo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Various payload types for different events
type PushPayload struct {
	Size    int      `json:"size"`
	Commits []Commit `json:"commits"`
	Ref     string   `json:"ref"`
}

type CreatePayload struct {
	Ref         string `json:"ref"`
	RefType     string `json:"ref_type"`
	Description string `json:"description"`
}

type IssuesPayload struct {
	Action string `json:"action"`
	Issue  struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
	} `json:"issue"`
}

type PullRequestPayload struct {
	Action      string `json:"action"`
	PullRequest struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
	} `json:"pull_request"`
}

type ForkPayload struct {
	Forkee struct {
		FullName string `json:"full_name"`
	} `json:"forkee"`
}

type ReleasePayload struct {
	Action  string `json:"action"`
	Release struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	} `json:"release"`
}

// Commit represents a git commit
type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
}

// Domain Business Logic

// GetBranch extracts the branch name from a git ref
func (p *PushPayload) GetBranch() string {
	return strings.TrimPrefix(p.Ref, "refs/heads/")
}

// GetShortSHA returns the first 7 characters of the commit SHA
func (c *Commit) GetShortSHA() string {
	if len(c.SHA) >= 7 {
		return c.SHA[:7]
	}
	return c.SHA
}

// GetFirstLine returns the first line of the commit message
func (c *Commit) GetFirstLine() string {
	lines := strings.Split(c.Message, "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return c.Message
}

// TruncateMessage truncates a message to maxLength and adds ellipsis if needed
func TruncateMessage(message string, maxLength int) string {
	if len(message) <= maxLength {
		return message
	}
	return message[:maxLength-3] + "..."
}

// FormatDescription returns a human-readable description of the event
func (e *GitHubEvent) FormatDescription() string {
	repoName := e.Repo.Name

	switch EventType(e.Type) {
	case EventTypePush:
		var payload PushPayload
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			branch := payload.GetBranch()
			if payload.Size == 1 {
				return fmt.Sprintf("Pushed 1 commit to %s (branch: %s)", repoName, branch)
			}
			return fmt.Sprintf(
				"Pushed %d commits to %s (branch: %s)",
				payload.Size,
				repoName,
				branch,
			)
		}

	case EventTypeCreate:
		var payload CreatePayload
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			return fmt.Sprintf("Created %s '%s' in %s", payload.RefType, payload.Ref, repoName)
		}

	case EventTypeDelete:
		var payload CreatePayload // Same structure as CreateEvent
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			return fmt.Sprintf("Deleted %s '%s' in %s", payload.RefType, payload.Ref, repoName)
		}

	case EventTypeIssues:
		var payload IssuesPayload
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			return fmt.Sprintf("%s issue #%d in %s: %s",
				titleCase(payload.Action),
				payload.Issue.Number,
				repoName,
				payload.Issue.Title)
		}

	case EventTypePullRequest:
		var payload PullRequestPayload
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			return fmt.Sprintf("%s pull request #%d in %s: %s",
				titleCase(payload.Action),
				payload.PullRequest.Number,
				repoName,
				payload.PullRequest.Title)
		}

	case EventTypeWatch:
		return fmt.Sprintf("Starred %s", repoName)

	case EventTypeFork:
		var payload ForkPayload
		if err := json.Unmarshal(e.Payload, &payload); err == nil &&
			payload.Forkee.FullName != "" {
			return fmt.Sprintf("Forked %s to %s", repoName, payload.Forkee.FullName)
		}
		return fmt.Sprintf("Forked %s", repoName)

	case EventTypeIssueComment:
		var payload IssuesPayload // Similar structure
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			return fmt.Sprintf("Commented on issue #%d in %s", payload.Issue.Number, repoName)
		}
		return fmt.Sprintf("Commented on an issue in %s", repoName)

	case EventTypePublic:
		return fmt.Sprintf("Made %s public", repoName)

	case EventTypeMember:
		return fmt.Sprintf("Added a member to %s", repoName)

	case EventTypeRelease:
		var payload ReleasePayload
		if err := json.Unmarshal(e.Payload, &payload); err == nil {
			return fmt.Sprintf("Released %s in %s", payload.Release.TagName, repoName)
		}
		return fmt.Sprintf("Created a release in %s", repoName)
	}

	return fmt.Sprintf("%s in %s", e.Type, repoName)
}

// GetCommitDetails extracts commit information from a PushEvent
func (e *GitHubEvent) GetCommitDetails() ([]Commit, error) {
	if EventType(e.Type) != EventTypePush {
		return nil, fmt.Errorf("event is not a PushEvent")
	}

	var payload PushPayload
	if err := json.Unmarshal(e.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse push payload: %w", err)
	}

	return payload.Commits, nil
}

// titleCase converts the first letter to uppercase
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + strings.ToLower(s[1:])
}

// EventFilter represents filtering criteria for events
type EventFilter struct {
	Type     string
	MaxLimit int
}

// Matches checks if an event matches the filter criteria
func (f *EventFilter) Matches(event GitHubEvent) bool {
	if f.Type != "" && !strings.EqualFold(event.Type, f.Type) {
		return false
	}
	return true
}

// GetAvailableEventTypes returns all available event types with descriptions
func GetAvailableEventTypes() map[EventType]string {
	return map[EventType]string{
		EventTypePush:         "Git push",
		EventTypeCreate:       "Branch or tag creation",
		EventTypeDelete:       "Branch or tag deletion",
		EventTypeIssues:       "Issue opened, closed, etc.",
		EventTypePullRequest:  "PR opened, closed, merged, etc.",
		EventTypeWatch:        "Repository starred",
		EventTypeFork:         "Repository forked",
		EventTypeIssueComment: "Comment on issue or PR",
		EventTypePublic:       "Repository made public",
		EventTypeMember:       "Member added to repository",
		EventTypeRelease:      "Release published",
	}
}
