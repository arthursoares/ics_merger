# Calendar Merger Project Guidelines

## Commands
- **Build**: `go build -o bin/ical_merger ./cmd`
- **Run**: `./bin/ical_merger`
- **Run with local files**: `./bin/ical_merger -local -calendar-dir=./calendars`
- **Run HTTP server**: `./bin/ical_merger -serve`
- **Docker build**: `docker build -t ical_merger .`
- **Docker run**: `docker-compose up -d`
- **Run Go tests**: `make test`
- **Run Ruby compatibility tests**: `make test-ruby`
- **Run all tests**: `make test-all`
- **Run tests in Docker**: `./run-tests.sh`

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

## Project Status Summary

### Project Overview
We're working on an `ical_merger` Go application that:
- Merges multiple iCal calendars and serves them via HTTP
- Provides compatibility with Ruby iCalendar parsers
- Offers a TRMNL plugin interface for displaying calendar data

### Completed Changes

#### Ruby iCalendar Parser Compatibility
1. Created `RubyCompatibilityFixer` function in `internal/ical/ruby_compatibility.go`
2. Fixed date formatting with `VALUE=DATE` for all-day events and `TZID` for timed events
3. Added CALSCALE and VTIMEZONE components required by Ruby
4. Integrated fixes into HTTP handlers and merged output

#### Custom Timezone Support
1. Added `OutputTimezone` field to Config struct
2. Updated docker-compose.yml with `OUTPUT_TIMEZONE` environment variable
3. Applied configured timezone to all calendar outputs

#### TRMNL Plugin Templates
1. Updated `trmnl-calendar-template.liquid` and `trmnl-pixel-calendar-template.liquid`
2. Optimized for TRMNL display dimensions (384×192px)
3. Matched reference styling with NicoPups/NicoClean fonts, pixelated borders
4. Added proper event indexing and all-day event representation

#### Testing
1. Added comprehensive testing with Ruby validation
2. Created Docker-based testing environment (`Dockerfile.test`)
3. Added Makefile targets for validation (`test-ruby`, `test-all`)

### Current Status
- All code changes have been committed to Git
- We have functioning Ruby-compatible iCal output
- The TRMNL templates now match the reference design
- Automated testing is in place for validation

### Files Modified
- `/internal/ical/ruby_compatibility.go` - New file for Ruby compatibility
- `/internal/app/merger.go` - Updated to apply Ruby fixes
- `/cmd/ical_merger.go` - Updated HTTP handlers
- `/trmnl-calendar-template.liquid` - Updated to match reference design
- `/trmnl-pixel-calendar-template.liquid` - Simplified vertical layout version
- `/tests/*` - Ruby validation test scripts
- `Dockerfile.test` and `run-tests.sh` - Testing infrastructure