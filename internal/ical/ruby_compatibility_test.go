package ical

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRubyCompatibilityFixer tests that the RubyCompatibilityFixer function produces
// valid calendar data that can be parsed by the Ruby icalendar gem.
func TestRubyCompatibilityFixer(t *testing.T) {
	// Skip this test if Ruby is not installed
	if _, err := exec.LookPath("ruby"); err != nil {
		t.Skip("Ruby not installed, skipping test")
	}

	// Create a test calendar with various date formats
	testCalendar := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//ical_merger//TEST//EN
METHOD:PUBLISH
BEGIN:VEVENT
UID:test-event-1
SUMMARY:All-day Event
DTSTART:20250101
DTEND:20250102
END:VEVENT
BEGIN:VEVENT
UID:test-event-2
SUMMARY:Timed Event
DTSTART:20250101T120000
DTEND:20250101T130000
END:VEVENT
BEGIN:VEVENT
UID:test-event-3
SUMMARY:Event with TZID
DTSTART;TZID=Europe/Berlin:20250101T120000
DTEND;TZID=Europe/Berlin:20250101T130000
END:VEVENT
BEGIN:VEVENT
UID:test-event-4
SUMMARY:Event with VALUE=DATE
DTSTART;VALUE=DATE:20250101
DTEND;VALUE=DATE:20250102
END:VEVENT
BEGIN:VEVENT
UID:test-event-5
SUMMARY:Event with malformed TZID
DTEND;TZID=Europe/Berlin:;TZID=Europe/Berlin:20250101T130000
END:VEVENT
END:VCALENDAR`

	// Create temp directory for test files
	tempDir, err := ioutil.TempDir("", "ical-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create input file
	inputPath := filepath.Join(tempDir, "input.ics")
	if err := ioutil.WriteFile(inputPath, []byte(testCalendar), 0644); err != nil {
		t.Fatalf("Failed to write test calendar: %v", err)
	}

	// Apply fixes
	fixedCalendar := RubyCompatibilityFixer(testCalendar, "Europe/Berlin")

	// Create output file
	outputPath := filepath.Join(tempDir, "output.ics")
	if err := ioutil.WriteFile(outputPath, []byte(fixedCalendar), 0644); err != nil {
		t.Fatalf("Failed to write fixed calendar: %v", err)
	}

	// Validate with Ruby
	rubyScript := `
#!/usr/bin/env ruby
require 'icalendar'

begin
  cal = Icalendar::Calendar.parse(File.read(ARGV[0]))
  if cal.empty? || cal.first.events.empty?
    puts "ERROR: No events found"
    exit(1)
  end
  puts "OK: Parsed #{cal.first.events.size} events"
  exit(0)
rescue => e
  puts "ERROR: #{e.message}"
  exit(1)
end
`

	// Write Ruby script to temp file
	scriptPath := filepath.Join(tempDir, "validate.rb")
	if err := ioutil.WriteFile(scriptPath, []byte(rubyScript), 0755); err != nil {
		t.Fatalf("Failed to write Ruby script: %v", err)
	}

	// Run Ruby validation
	cmd := exec.Command("ruby", scriptPath, outputPath)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("Ruby validation failed: %v\nOutput: %s", err, output)
	}
	
	// Check for success
	if !strings.Contains(string(output), "OK") {
		t.Errorf("Validation output does not contain OK: %s", output)
	}
}