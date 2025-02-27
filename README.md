# iCal Merger

A tool to combine multiple iCalendar (.ics) files into a single unified calendar.

## Features

- Merges multiple iCalendar sources into a single .ics file
- Handles duplicate events across calendars
- Prepends calendar name to event titles from single-source events
- Periodically updates the merged calendar
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

### Running Locally

```
go build -o bin/ical_merger ./cmd
./bin/ical_merger
```

Or use the Makefile:

```
make run
```

## How It Works

1. The app fetches each calendar from the provided URLs
2. It identifies duplicate events by comparing event summaries
3. For events that appear in only one calendar, it prepends the calendar name in square brackets
4. For events that appear in multiple calendars, it keeps the original title
5. The merged calendar is saved to the configured output path
6. The process repeats at the configured interval

## License

MIT