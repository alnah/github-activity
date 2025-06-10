package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// CLI Layer - User interface and presentation

// CLI handles command-line interface
type CLI struct {
	service *ActivityService
	output  OutputFormatter
}

// NewCLI creates a new CLI instance
func NewCLI(service *ActivityService) *CLI {
	return &CLI{
		service: service,
		output:  &ConsoleOutputFormatter{},
	}
}

// CLIFlags represents command-line flags
type CLIFlags struct {
	EventType string
	Limit     int
	Detailed  bool
	ListTypes bool
	Args      []string // Non-flag arguments
}

// Run executes the CLI
func (c *CLI) Run(args []string) int {
	flags := c.parseFlags(args)

	// Handle list-types flag
	if flags.ListTypes {
		c.listEventTypes()
		return 0
	}

	// Check if username is provided
	if len(flags.Args) < 1 {
		c.printUsage()
		return 1
	}

	username := flags.Args[0]

	// Create filter
	filter := EventFilter{
		Type:     flags.EventType,
		MaxLimit: flags.Limit,
	}

	// Validate options
	options := ActivityOptions{
		EventType:    flags.EventType,
		Limit:        flags.Limit,
		ShowDetailed: flags.Detailed,
	}

	if err := options.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Fetch and display activities
	fmt.Printf("Fetching GitHub activity for user: %s\n\n", username)

	if flags.Detailed {
		return c.displayDetailedActivities(username, filter)
	}

	return c.displayActivities(username, filter)
}

// parseFlags parses command-line flags
func (c *CLI) parseFlags(args []string) CLIFlags {
	flags := CLIFlags{}

	flagSet := flag.NewFlagSet("github-activity", flag.ContinueOnError)
	flagSet.StringVar(
		&flags.EventType,
		"type",
		"",
		"Filter by event type (e.g., PushEvent, IssuesEvent)",
	)
	flagSet.IntVar(&flags.Limit, "limit", 30, "Limit the number of events displayed")
	flagSet.BoolVar(&flags.Detailed, "detailed", false, "Show detailed information for each event")
	flagSet.BoolVar(&flags.ListTypes, "list-types", false, "List all available event types")

	flagSet.Usage = c.printUsage

	// Parse flags
	if err := flagSet.Parse(args[1:]); err != nil {
		// Don't exit here for testing
		return flags
	}

	// Store remaining arguments
	flags.Args = flagSet.Args()

	return flags
}

// displayActivities displays activities in summary format
func (c *CLI) displayActivities(username string, filter EventFilter) int {
	activities, err := c.service.GetUserActivity(username, filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(activities) == 0 {
		if filter.Type != "" {
			fmt.Printf("No '%s' events found.\n", filter.Type)
		} else {
			fmt.Println("No recent activity found.")
		}
		return 0
	}

	c.output.FormatActivities(os.Stdout, activities)
	return 0
}

// displayDetailedActivities displays activities with detailed information
func (c *CLI) displayDetailedActivities(username string, filter EventFilter) int {
	activities, err := c.service.GetUserActivityDetailed(username, filter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if len(activities) == 0 {
		if filter.Type != "" {
			fmt.Printf("No '%s' events found.\n", filter.Type)
		} else {
			fmt.Println("No recent activity found.")
		}
		return 0
	}

	c.output.FormatDetailedActivities(os.Stdout, activities)
	return 0
}

// listEventTypes displays available event types
func (c *CLI) listEventTypes() {
	eventTypes := GetAvailableEventTypes()

	fmt.Println("Available event types:")
	maxTypeLen := 0
	for eventType := range eventTypes {
		if len(eventType) > maxTypeLen {
			maxTypeLen = len(eventType)
		}
	}

	for eventType, description := range eventTypes {
		fmt.Printf("  %-*s - %s\n", maxTypeLen+2, eventType, description)
	}
}

// printUsage prints usage information
func (c *CLI) printUsage() {
	fmt.Println("GitHub Activity CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  github-activity [flags] <username>")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -type string")
	fmt.Println("        Filter by event type (e.g., PushEvent, IssuesEvent)")
	fmt.Println("  -limit int")
	fmt.Println("        Limit the number of events displayed (default 30)")
	fmt.Println("  -detailed")
	fmt.Println("        Show detailed information for each event")
	fmt.Println("  -list-types")
	fmt.Println("        List all available event types")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  github-activity kamranahmedse")
	fmt.Println("  github-activity -type=PushEvent -limit=5 torvalds")
	fmt.Println("  github-activity -detailed octocat")
	fmt.Println("  github-activity -list-types")
}

// OutputFormatter interface for formatting output
type OutputFormatter interface {
	FormatActivities(w io.Writer, activities []ActivitySummary)
	FormatDetailedActivities(w io.Writer, activities []DetailedActivity)
}

// ConsoleOutputFormatter formats output for console
type ConsoleOutputFormatter struct{}

// FormatActivities formats activity summaries for console
func (f *ConsoleOutputFormatter) FormatActivities(w io.Writer, activities []ActivitySummary) {
	for _, activity := range activities {
		_, _ = fmt.Fprintf(w, "- %s\n", activity.Description)
	}
}

// FormatDetailedActivities formats detailed activities for console
func (f *ConsoleOutputFormatter) FormatDetailedActivities(
	w io.Writer,
	activities []DetailedActivity,
) {
	for _, activity := range activities {
		_, _ = fmt.Fprintf(w, "- %s\n", activity.Description)
		_, _ = fmt.Fprintf(w, "  Time: %s\n", activity.Timestamp)
		_, _ = fmt.Fprintf(w, "  Type: %s\n", activity.Type)

		// Show commits for push events
		if len(activity.Commits) > 0 {
			_, _ = fmt.Fprintln(w, "  Commits:")
			for _, commit := range activity.Commits {
				_, _ = fmt.Fprintf(w, "    - %s: %s\n", commit.SHA, commit.Message)
			}
		}

		// Show extra details if any
		for key, value := range activity.ExtraDetails {
			// Capitalize first letter of key
			capitalizedKey := key
			if len(key) > 0 {
				capitalizedKey = strings.ToUpper(key[:1]) + key[1:]
			}
			_, _ = fmt.Fprintf(w, "  %s: %s\n", capitalizedKey, value)
		}

		_, _ = fmt.Fprintln(w)
	}
}
