# Calendar Merger Project Guidelines

## Commands
- **Build**: `go build -o bin/ical_merger ./cmd`
- **Run**: `./bin/ical_merger`
- **Run with local files**: `./bin/ical_merger -local -calendar-dir=./calendars`
- **Run HTTP server**: `./bin/ical_merger -serve`
- **Docker build**: `docker build -t ical_merger .`
- **Docker run**: `docker-compose up -d`

## Coding Style
- **Formatting**: Format code with `gofmt`
- **Imports**: Group standard lib imports first, then external, then internal
- **Error handling**: Always check errors and return them to the caller
- **Type names**: Use PascalCase for public types, camelCase for private
- **Function names**: Use PascalCase for exported functions
- **Comments**: Public functions and types must have comments
- **Logging**: Use the standard `log` package for informational and error messages
- **Testing**: Place tests in the same package as the code being tested

Follow Go idiomatic patterns and the principles from [Effective Go](https://golang.org/doc/effective_go.html).