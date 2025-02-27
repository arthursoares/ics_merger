# iCal Merger

A tool to combine multiple iCalendar (.ics) files into a single unified calendar.

## Features

- Merges multiple iCalendar sources into a single .ics file
- Handles duplicate events across calendars
- Prepends calendar name to event titles from single-source events
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
  "syncIntervalMinutes": 15
}
```

### Running with Docker

Build and run using Docker Compose:

```
docker-compose up -d
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

## License

MIT