# GitHub Activity CLI

A command-line tool to fetch and display GitHub user activity using Clean Architecture principles.
Built for my educational purpose.

## Features

- Fetch recent GitHub activity for any user
- Filter activities by event type
- Display detailed information including commit messages
- Clean Architecture with separated domain, repository, application, and CLI layers
- Caching support to minimize API calls
- Comprehensive error handling

## Installation

```bash
# Clone the repository
git clone https://github.com/alnah/github-activity
cd github-activity-cli

# Build the application
go build -o github-activity

# Or install directly
go install
```

## Usage

### Basic Usage

```bash
# Fetch activity for a user
github-activity alnah

# Filter by event type
github-activity -type=PushEvent torvalds

# Limit number of events
github-activity -limit=5 octocat

# Show detailed information
github-activity -detailed octocat

# List available event types
github-activity -list-types
```

### Command-Line Flags

- `-type string`: Filter by event type (e.g., PushEvent, IssuesEvent)
- `-limit int`: Limit the number of events displayed (default: 30)
- `-detailed`: Show detailed information for each event
- `-list-types`: List all available event types

### Examples

```bash
# Show only push events for Linus Torvalds
github-activity -type=PushEvent -limit=10 torvalds

# Get detailed activity including commit messages
github-activity -detailed -limit=5 alnah

# See all available event types
github-activity -list-types
```

## Event Types

The tool supports all GitHub event types:

- **PushEvent**: Git push
- **CreateEvent**: Branch or tag creation
- **DeleteEvent**: Branch or tag deletion
- **IssuesEvent**: Issue opened, closed, etc.
- **PullRequestEvent**: PR opened, closed, merged, etc.
- **WatchEvent**: Repository starred
- **ForkEvent**: Repository forked
- **IssueCommentEvent**: Comment on issue or PR
- **PublicEvent**: Repository made public
- **MemberEvent**: Member added to repository
- **ReleaseEvent**: Release published

## Testing

The project includes comprehensive tests for each layer:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific layer
go test -v -run TestDomain
go test -v -run TestRepository
go test -v -run TestApplication
go test -v -run TestCLI

# Run benchmarks
go test -bench=.
```

## Design Decisions

### Clean Architecture

The project follows Clean Architecture principles to ensure:

- **Testability**: Each layer can be tested independently
- **Maintainability**: Clear separation of concerns
- **Flexibility**: Easy to swap implementations (e.g., different data sources)

### Caching

- API responses are cached for 5 minutes per user
- Reduces unnecessary API calls
- Improves response time for repeated queries

### Error Handling

- Structured errors with codes and messages
- User-friendly error messages
- Proper HTTP status code handling

## Development

### Adding New Features

1. **New Event Type Support**: Add parsing logic in `domain.go`
2. **New Filter Options**: Extend `EventFilter` in domain and update CLI
3. **New Output Formats**: Implement `OutputFormatter` interface

### Code Structure

```go
// Domain entities are immutable and contain business logic
type GitHubEvent struct {
    ID        string
    Type      string
    // ...
}

// Repository interface for data access
type EventRepository interface {
    FetchEvents(username string) ([]GitHubEvent, error)
}

// Application services orchestrate business logic
type ActivityService struct {
    repository EventRepository
}

// CLI handles user interaction
type CLI struct {
    service *ActivityService
    output  OutputFormatter
}
```

## Performance

- Caching reduces API calls
- Efficient filtering without loading all data
- Minimal memory footprint
- Fast response times

## Error Handling

The tool provides clear error messages for common scenarios:

- User not found
- Rate limit exceeded
- Network errors
- Invalid command-line arguments

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Inspired by [roadmap.sh GitHub Activity Project](https://roadmap.sh/projects/github-user-activity)
- Built with Go's standard library
- Uses GitHub's public API
