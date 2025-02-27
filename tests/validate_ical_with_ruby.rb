#!/usr/bin/env ruby
# Test script to validate iCal files with Ruby parser

require 'icalendar'
require 'fileutils'
require 'minitest/autorun'

class IcalValidationTest < Minitest::Test
  def setup
    # Calendar path - either provided as argument or using default merged output
    @calendar_path = ARGV[0] || File.expand_path("../output/merged.ics", File.dirname(__FILE__))
    puts "Testing calendar file: #{@calendar_path}"
    
    # Verify file exists
    assert File.exist?(@calendar_path), "Calendar file not found"
  end
  
  def test_calendar_parseable_by_ruby
    # Read the calendar file
    begin
      content = File.read(@calendar_path)
    rescue => e
      assert false, "Error reading calendar file: #{e.message}"
    end
    
    # Try to parse with the icalendar gem
    begin
      calendars = Icalendar::Calendar.parse(content)
      
      # Basic validation
      assert !calendars.empty?, "No calendars found in the file"
      assert_instance_of Icalendar::Calendar, calendars.first, "First object is not a calendar"
      
      # Event validation
      assert !calendars.first.events.empty?, "No events found in the calendar"
      puts "Successfully parsed calendar with #{calendars.first.events.size} events"
      
      # Check for required properties
      calendar = calendars.first
      assert calendar.prodid, "Calendar missing PRODID property"
      assert calendar.calscale, "Calendar missing CALSCALE property"
      
      # Check timezone
      tzid = nil
      calendar.components.each do |component|
        if component.class.to_s == "Icalendar::Timezone"
          tzid = component.tzid.to_s if component.tzid
          break
        end
      end
      assert tzid, "No VTIMEZONE component found with TZID"
      puts "Calendar has timezone: #{tzid}"
      
      # Check event formatting
      puts "Checking event date formatting..."
      all_day_count = 0
      timed_count = 0
      
      calendar.events.take(10).each do |event|
        if event.dtstart.to_ical.include?('T')  # If it has a time component
          timed_count += 1
          assert event.dtstart.ical_params['TZID'], 
                 "Missing TZID parameter for timed event: #{event.summary}"
        else
          all_day_count += 1
          assert event.dtstart.ical_params['VALUE'] == ['DATE'], 
                 "Missing VALUE=DATE parameter for all-day event: #{event.summary}"
        end
        
        if event.dtend
          if event.dtend.to_ical.include?('T')  # If it has a time component
            assert event.dtend.ical_params['TZID'], 
                   "Missing TZID parameter for timed event end: #{event.summary}"
          else
            assert event.dtend.ical_params['VALUE'] == ['DATE'], 
                   "Missing VALUE=DATE parameter for all-day event end: #{event.summary}"
          end
        end
      end
      
      puts "Checked events: #{all_day_count} all-day, #{timed_count} timed events"
      
    rescue => e
      assert false, "Error parsing calendar: #{e.message}\n#{e.backtrace.join("\n")}"
    end
  end
end

# Only run tests if executed directly
if __FILE__ == $PROGRAM_NAME
  Minitest.run