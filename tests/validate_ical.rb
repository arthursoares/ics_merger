#!/usr/bin/env ruby
# Tests that verify the iCalendar output is valid with Ruby ical parser

require 'minitest/autorun'
require 'icalendar'
require 'json'
require 'fileutils'

class IcalValidationTest < Minitest::Test
  def setup
    # Path to our test outputs
    @output_dir = File.expand_path("../output", File.dirname(__FILE__))
    @bin_dir = File.expand_path("../bin", File.dirname(__FILE__))
    
    # Make sure output directory exists
    FileUtils.mkdir_p(@output_dir) unless Dir.exist?(@output_dir)
    
    # Build latest version
    system("cd #{File.dirname(@output_dir)} && go build -o bin/ical_merger ./cmd")
    
    # Create test calendar if it doesn't exist
    unless File.exist?(File.join(@output_dir, "test_calendar.ics"))
      generate_test_calendar
    end
  end
  
  def generate_test_calendar
    # Run the merger in local mode with test calendar data
    test_config = {
      calendars: [
        { name: "Test", url: "file://#{@bin_dir}/test.ics" }
      ],
      outputPath: "#{@output_dir}/test_calendar.ics",
      syncIntervalMinutes: 15,
      outputTimezone: "Europe/Berlin"
    }
    
    # Create test calendar data
    test_cal = <<~ICAL
      BEGIN:VCALENDAR
      VERSION:2.0
      PRODID:-//Test Calendar//EN
      BEGIN:VEVENT
      UID:test-event-1
      SUMMARY:Test All-day Event
      DTSTART:20250101
      DTEND:20250102
      END:VEVENT
      BEGIN:VEVENT
      UID:test-event-2
      SUMMARY:Test Timed Event
      DTSTART:20250101T120000
      DTEND:20250101T130000
      END:VEVENT
      END:VCALENDAR
    ICAL
    
    # Write test calendar
    File.write(File.join(@bin_dir, "test.ics"), test_cal)
    
    # Write test config
    File.write(File.join(@output_dir, "test_config.json"), JSON.pretty_generate(test_config))
    
    # Run merger with test config
    system("#{@bin_dir}/ical_merger -config #{@output_dir}/test_config.json")
  end
  
  def test_calendar_parseable_by_ruby
    # Path to the output file
    ical_file = File.join(@output_dir, "test_calendar.ics")
    
    # Check that file exists
    assert File.exist?(ical_file), "iCal output file not found"
    
    # Try to parse with the icalendar gem
    begin
      calendars = Icalendar::Calendar.parse(File.read(ical_file))
      
      # Check that at least one calendar was parsed
      assert !calendars.empty?, "No calendars found in the output"
      
      # Check the calendar has events
      assert !calendars.first.events.empty?, "No events found in the calendar"
      
      # Test that the required properties are present
      calendar = calendars.first
      assert calendar.prodid, "Missing PRODID property"
      assert calendar.calscale, "Missing CALSCALE property"
      
      # Check that time zones are correctly set for all events
      calendars.first.events.each do |event|
        if event.dtstart.to_ical.include?('T')  # If it has a time component (not an all-day event)
          assert event.dtstart.ical_params["TZID"], "Missing TZID parameter for event start time: #{event.summary}"
        elsif event.dtstart
          assert event.dtstart.ical_params["VALUE"] == ["DATE"], "Missing VALUE=DATE parameter for all-day event: #{event.summary}"
        end
        
        if event.dtend && event.dtend.to_ical.include?('T')
          assert event.dtend.ical_params["TZID"], "Missing TZID parameter for event end time: #{event.summary}"
        elsif event.dtend
          assert event.dtend.ical_params["VALUE"] == ["DATE"], "Missing VALUE=DATE parameter for all-day event end: #{event.summary}"
        end
      end
      
    rescue => e
      # If there's an error, the test fails
      assert false, "Error parsing calendar: #{e.message}\n#{e.backtrace.join("\n")}"
    end
  end
  
  def test_ruby_script_validation
    # Path to the output file
    ical_file = File.join(@output_dir, "test_calendar.ics")
    debug_script = File.join(File.dirname(@output_dir), "debug_ical_parser.rb")
    
    # Run the debug script on our test calendar
    result = `ruby #{debug_script} #{ical_file}`
    
    # Check that the script succeeded in parsing
    assert result.include?("Success!"), "Debug script failed to parse the calendar: #{result}"
  end
end