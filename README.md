# iCal Merger

A tool to combine multiple iCalendar (.ics) files into a single unified calendar.

## Features

- Merges multiple iCalendar sources into a single .ics file
- Handles duplicate events across calendars
- Prepends calendar name to event titles from single-source events (e.g., "[Work] Meeting")
- Maintains original titles for events that appear in multiple calendars
- Robust support for Apple Calendar/iCloud exports
- Periodically updates the merged calendar
- HTTP server to serve the merged calendar
- Support for local files or remote URLs
- Docker containerized for easy deployment

## Getting Started

### Configuration

Copy the example configuration file:

```
cp config.json.example config.json
```

Edit `config.json` to add your calendar sources:

```json
{
  "calendars": [
    {
      "name": "Home",
      "url": "https://example.com/home.ics"
    },
    {
      "name": "Arthur",
      "url": "https://example.com/arthur.ics"
    },
    {
      "name": "Hannah",
      "url": "https://example.com/hannah.ics"
    }
  ],
  "outputPath": "/app/output/merged.ics",
  "syncIntervalMinutes": 15,
  "outputTimezone": "Europe/Berlin"
}
```

### Running with Docker

Build and run using Docker Compose:

```
docker-compose up -d
```

You can set a custom timezone using an environment variable:

```
OUTPUT_TIMEZONE=America/New_York docker-compose up -d
```

Or use the provided Makefile:

```
make docker-build
make docker-run
```

Then access the merged calendar at `http://localhost:8080/calendar`.

### Running Locally

With URL sources:

```
go build -o bin/ical_merger ./cmd
./bin/ical_merger
```

With local files:

```
go build -o bin/ical_merger ./cmd
./bin/ical_merger -local -calendar-dir=/path/to/calendars
```

As an HTTP server:

```
./bin/ical_merger -serve -addr=:8080
```

## Usage Options

```
Usage of ical_merger:
  -addr string
        HTTP server address (default ":8080")
  -calendar-dir string
        Directory containing calendar files when using local mode (default "./calendars")
  -config string
        Path to config file (overrides CONFIG_PATH env var)
  -local
        Run with local files instead of URLs
  -output string
        Path to output file (overrides config)
  -serve
        Run as HTTP server
```

## Accessing the Calendar

When running in HTTP server mode, the following endpoints are available:

- `/calendar` - Get the merged calendar file
- `/health` - Health check endpoint

## How It Works

1. The app fetches each calendar from the provided URLs or local files
2. It identifies duplicate events by comparing event summaries
3. For events that appear in only one calendar, it prepends the calendar name in square brackets
4. For events that appear in multiple calendars, it keeps the original title
5. The merged calendar is saved to the configured output path and/or served via HTTP
6. The process repeats at the configured interval

### Title Modification Details

The merged calendar modifies event titles to help you identify which calendar they came from:

- **Single-source events**: Events that appear in only one calendar will have the calendar name prepended in square brackets.
  
  Example: If an event called "Dinner with friends" appears only in your "Personal" calendar, it will appear as "[Personal] Dinner with friends" in the merged calendar.

- **Multi-source events**: Events with identical summaries that appear in multiple calendars will keep their original title without modification, as they are recognized as the same event.
  
  Example: If an event called "Company Meeting" appears in both your "Work" and "Team" calendars, it will remain as "Company Meeting" in the merged calendar.

### Output Calendar Format

The merged calendar maintains all essential event properties while ensuring compatibility:

- **Standard iCalendar format**: The output is a standard iCal (.ics) file that can be imported into any calendar application
- **Preserved properties**: Event dates, times, locations, descriptions, and other properties are preserved
- **Event UIDs**: Each event maintains its original UID to avoid duplication when importing
- **Timezone handling**: The calendar preserves timezone information from the source calendars

Sample event entry in the merged calendar:

```
BEGIN:VEVENT
UID:12345-67890-ABCDEF
SUMMARY:[Work] Project Meeting
DTSTART:20240328T140000
DTEND:20240328T150000
LOCATION:Conference Room B
DESCRIPTION:Weekly project status meeting
END:VEVENT
```

## Testing

The project includes automated tests for both Go code and Ruby iCal format compatibility:

```
# Run Go tests
make test

# Run Ruby icalendar compatibility tests
make test-ruby

# Run all tests
make test-all
```

### Docker Testing

You can also run tests in a Docker container using the provided script:

```
./run-tests.sh
```

This builds a special test container with both Go and Ruby installed and runs all tests to ensure compatibility.

## Configuration Options

### Output Timezone

You can specify the timezone used for calendar output:

- In `config.json` with the `outputTimezone` property
- Via environment variable `OUTPUT_TIMEZONE` (when using Docker)
- Default is `Europe/Berlin` if not specified

The timezone is used for:
- All timed events (setting the TZID parameter)
- The VTIMEZONE component in the calendar
- The X-WR-TIMEZONE property

## License

MIT

## Credits

This project's implementation was developed with assistance from Claude Code, which helped create a robust parser with special support for Apple Calendar/iCloud exports and the event merging logic.