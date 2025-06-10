package main

import (
	"fmt"
	"strings"
)

// Application Service Layer - Business logic and use cases

// ActivityService handles the business logic for GitHub activities
type ActivityService struct {
	repository EventRepository
}

// NewActivityService creates a new activity service
func NewActivityService(repository EventRepository) *ActivityService {
	return &ActivityService{
		repository: repository,
	}
}

// GetUserActivity fetches and filters user activities
func (s *ActivityService) GetUserActivity(
	username string,
	filter EventFilter,
) ([]ActivitySummary, error) {
	// Validate username
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// Fetch events from repository
	events, err := s.repository.FetchEvents(username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	// Apply filtering and convert to summaries
	summaries := make([]ActivitySummary, 0)
	count := 0

	for _, event := range events {
		// Apply filter
		if !filter.Matches(event) {
			continue
		}

		// Check limit
		if filter.MaxLimit > 0 && count >= filter.MaxLimit {
			break
		}

		// Create summary
		summary := s.createActivitySummary(event)
		summaries = append(summaries, summary)
		count++
	}

	return summaries, nil
}

// GetUserActivityDetailed fetches activities with detailed information
func (s *ActivityService) GetUserActivityDetailed(
	username string,
	filter EventFilter,
) ([]DetailedActivity, error) {
	// Validate username
	if strings.TrimSpace(username) == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// Fetch events
	events, err := s.repository.FetchEvents(username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	// Apply filtering and create detailed activities
	activities := make([]DetailedActivity, 0)
	count := 0

	for _, event := range events {
		// Apply filter
		if !filter.Matches(event) {
			continue
		}

		// Check limit
		if filter.MaxLimit > 0 && count >= filter.MaxLimit {
			break
		}

		// Create detailed activity
		activity := s.createDetailedActivity(event)
		activities = append(activities, activity)
		count++
	}

	return activities, nil
}

// ActivitySummary represents a summarized view of an activity
type ActivitySummary struct {
	Description string
	Type        string
	Repository  string
	Timestamp   string
}

// DetailedActivity represents a detailed view of an activity
type DetailedActivity struct {
	ActivitySummary
	EventID      string
	ActorLogin   string
	CommitCount  int
	Commits      []CommitSummary
	ExtraDetails map[string]string
}

// CommitSummary represents a simplified commit
type CommitSummary struct {
	SHA     string
	Message string
	Author  string
}

// createActivitySummary creates a summary from an event
func (s *ActivityService) createActivitySummary(event GitHubEvent) ActivitySummary {
	return ActivitySummary{
		Description: event.FormatDescription(),
		Type:        event.Type,
		Repository:  event.Repo.Name,
		Timestamp:   event.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// createDetailedActivity creates a detailed activity from an event
func (s *ActivityService) createDetailedActivity(event GitHubEvent) DetailedActivity {
	activity := DetailedActivity{
		ActivitySummary: s.createActivitySummary(event),
		EventID:         event.ID,
		ActorLogin:      event.Actor.Login,
		ExtraDetails:    make(map[string]string),
	}

	// Add type-specific details
	switch EventType(event.Type) {
	case EventTypePush:
		commits, err := event.GetCommitDetails()
		if err == nil {
			activity.CommitCount = len(commits)
			for _, commit := range commits {
				activity.Commits = append(activity.Commits, CommitSummary{
					SHA:     commit.GetShortSHA(),
					Message: TruncateMessage(commit.GetFirstLine(), 60),
					Author:  commit.Author.Name,
				})
			}
		}
	}

	return activity
}

// GetEventTypeStatistics returns statistics about event types
func (s *ActivityService) GetEventTypeStatistics(username string) (map[string]int, error) {
	events, err := s.repository.FetchEvents(username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	stats := make(map[string]int)
	for _, event := range events {
		stats[event.Type]++
	}

	return stats, nil
}

// GetRecentRepositories returns a list of recently active repositories
func (s *ActivityService) GetRecentRepositories(username string, limit int) ([]string, error) {
	events, err := s.repository.FetchEvents(username)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	// Use a map to track unique repositories while preserving order
	repoMap := make(map[string]bool)
	repos := make([]string, 0)

	for _, event := range events {
		if !repoMap[event.Repo.Name] {
			repoMap[event.Repo.Name] = true
			repos = append(repos, event.Repo.Name)

			if limit > 0 && len(repos) >= limit {
				break
			}
		}
	}

	return repos, nil
}

// ActivityOptions represents options for activity queries
type ActivityOptions struct {
	EventType    string
	Limit        int
	ShowDetailed bool
}

// DefaultActivityOptions returns default options
func DefaultActivityOptions() ActivityOptions {
	return ActivityOptions{
		EventType:    "",
		Limit:        30,
		ShowDetailed: false,
	}
}

// Validate validates the activity options
func (o *ActivityOptions) Validate() error {
	if o.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if o.EventType != "" {
		// Validate event type
		validTypes := GetAvailableEventTypes()
		found := false
		for eventType := range validTypes {
			if strings.EqualFold(o.EventType, string(eventType)) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid event type: %s", o.EventType)
		}
	}

	return nil
}
