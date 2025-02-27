#!/usr/bin/env ruby
# This script generates a test calendar with various date formats
# to verify that the merger handles them correctly

require 'fileutils'

def generate_test_calendar(output_path)
  content = <<~ICAL
    BEGIN:VCALENDAR
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
    DTSTART;TZID=Europe/Berlin:;TZID=Europe/Berlin:20250101T120000
    END:VEVENT
    BEGIN:VEVENT
    UID:test-event-6
    SUMMARY:Event with no DTEND
    DTSTART:20250101T120000
    END:VEVENT
    END:VCALENDAR
  ICAL
  
  # Ensure directory exists
  FileUtils.mkdir_p(File.dirname(output_path))
  
  # Write test calendar
  File.write(output_path, content)
  puts "Generated test calendar at: #{output_path}"
end

# Main
if __FILE__ == $0
  output_path = ARGV[0] || "calendars/test.ics"
  generate_test_calendar(output_path)
end