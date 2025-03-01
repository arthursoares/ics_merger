package ical

import (
	"fmt"
	"strings"
)

// RubyCompatibilityFixer ensures the calendar output is compatible with the Ruby iCalendar parser
func RubyCompatibilityFixer(icalData string, timezone string) string {
	// 1. Normalize line endings
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(icalData, "\r\n", "\n"), "\r", "\n"), "\n")
	
	// 2. Initialize a new calendar with all required elements
	var output []string
	output = append(output, "BEGIN:VCALENDAR")
	output = append(output, "VERSION:2.0")
	output = append(output, "CALSCALE:GREGORIAN")
	output = append(output, "METHOD:PUBLISH")
	output = append(output, "PRODID:-//ical_merger//RUBY_COMPAT//EN")
	output = append(output, fmt.Sprintf("X-WR-CALNAME:Merged Calendar"))
	output = append(output, fmt.Sprintf("X-WR-TIMEZONE:%s", timezone))
	
	// 3. Add VTIMEZONE component that Ruby parser expects
	output = append(output, "BEGIN:VTIMEZONE")
	output = append(output, fmt.Sprintf("TZID:%s", timezone))
	output = append(output, "BEGIN:STANDARD")
	output = append(output, "DTSTART:19701101T030000")
	output = append(output, "TZOFFSETFROM:+0200")
	output = append(output, "TZOFFSETTO:+0100")
	output = append(output, "RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU")
	output = append(output, "END:STANDARD")
	output = append(output, "BEGIN:DAYLIGHT")
	output = append(output, "DTSTART:19700329T020000")
	output = append(output, "TZOFFSETFROM:+0100")
	output = append(output, "TZOFFSETTO:+0200")
	output = append(output, "RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU")
	output = append(output, "END:DAYLIGHT")
	output = append(output, "END:VTIMEZONE")
	
	// 4. Extract and fix events
	var inEvent bool
	var currentEvent []string
	var eventCount int
	
	for _, line := range lines {
		if line == "BEGIN:VEVENT" {
			inEvent = true
			currentEvent = []string{"BEGIN:VEVENT"}
		} else if line == "END:VEVENT" && inEvent {
			currentEvent = append(currentEvent, "END:VEVENT")
			
			// Process the event and add to output if valid
			if fixedEvent := fixEvent(currentEvent, timezone); fixedEvent != nil {
				output = append(output, fixedEvent...)
				eventCount++
			}
			
			inEvent = false
			currentEvent = nil
		} else if inEvent {
			currentEvent = append(currentEvent, line)
		}
	}
	
	// 5. End the calendar
	output = append(output, "END:VCALENDAR")
	
	return strings.Join(output, "\n")
}

// fixEvent processes a single event and returns it in Ruby-compatible format
func fixEvent(event []string, timezone string) []string {
	// Extract key properties
	var uid, summary, location, description string
	var dtstart, dtend string
	var dtStartParams, dtEndParams map[string]string
	var isAllDay bool
	var rrule string
	
	// Initialize parameter maps
	dtStartParams = make(map[string]string)
	dtEndParams = make(map[string]string)
	
	// Other properties to keep
	var otherProps []string
	
	for _, line := range event {
		if strings.HasPrefix(line, "UID:") {
			uid = strings.TrimPrefix(line, "UID:")
		} else if strings.HasPrefix(line, "SUMMARY:") {
			summary = strings.TrimPrefix(line, "SUMMARY:")
		} else if strings.HasPrefix(line, "LOCATION:") {
			location = strings.TrimPrefix(line, "LOCATION:")
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			description = strings.TrimPrefix(line, "DESCRIPTION:")
		} else if strings.HasPrefix(line, "RRULE:") {
			rrule = strings.TrimPrefix(line, "RRULE:")
		} else if strings.HasPrefix(line, "DTSTART") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dtstart = parts[1]
				
				// Check for parameters
				if strings.Contains(parts[0], ";") {
					paramParts := strings.SplitN(parts[0], ";", 2)
					if len(paramParts) > 1 {
						params := strings.Split(paramParts[1], ";")
						for _, param := range params {
							if strings.Contains(param, "=") {
								kv := strings.SplitN(param, "=", 2)
								if len(kv) == 2 {
									dtStartParams[kv[0]] = kv[1]
								}
							}
						}
					}
				}
				
				// Check if this is an all-day event
				if value, hasValueDate := dtStartParams["VALUE"]; hasValueDate && value == "DATE" {
					isAllDay = true
				} else if !strings.Contains(dtstart, "T") {
					// Also consider YYYYMMDD format as all-day event
					isAllDay = true
					dtStartParams["VALUE"] = "DATE"
				}
			}
		} else if strings.HasPrefix(line, "DTEND") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dtend = parts[1]
				
				// Check for parameters
				if strings.Contains(parts[0], ";") {
					paramParts := strings.SplitN(parts[0], ";", 2)
					if len(paramParts) > 1 {
						params := strings.Split(paramParts[1], ";")
						for _, param := range params {
							if strings.Contains(param, "=") {
								kv := strings.SplitN(param, "=", 2)
								if len(kv) == 2 {
									dtEndParams[kv[0]] = kv[1]
								}
							}
						}
					}
				}
			}
		} else if !strings.HasPrefix(line, "BEGIN:") && !strings.HasPrefix(line, "END:") {
			// Keep any other properties
			otherProps = append(otherProps, line)
		}
	}
	
	// Skip events without UID or DTSTART
	if uid == "" || dtstart == "" {
		return nil
	}
	
	// Build fixed event
	var fixedEvent []string
	fixedEvent = append(fixedEvent, "BEGIN:VEVENT")
	fixedEvent = append(fixedEvent, fmt.Sprintf("UID:%s", uid))
	
	if summary != "" {
		fixedEvent = append(fixedEvent, fmt.Sprintf("SUMMARY:%s", summary))
	}
	
	// Fix DTSTART format
	if isAllDay {
		// Ensure VALUE=DATE parameter for all-day events
		fixedEvent = append(fixedEvent, fmt.Sprintf("DTSTART;VALUE=DATE:%s", dtstart))
	} else {
		// Add TZID for timed events if missing
		fixedEvent = append(fixedEvent, fmt.Sprintf("DTSTART;TZID=%s:%s", timezone, dtstart))
	}
	
	// Fix DTEND format
	if dtend != "" {
		// Remove any malformed parameter patterns that might exist (like ;TZID=...:;TZID=...)
		if strings.HasPrefix(dtend, ";") {
			// Handle case like ";TZID=Europe/Berlin:20250213T154500"
			colonPos := strings.Index(dtend, ":")
			if colonPos > 0 {
				// Keep only what's after the colon
				dtend = dtend[colonPos+1:]
			}
		}
		
		if isAllDay {
			// Ensure VALUE=DATE parameter for all-day events
			fixedEvent = append(fixedEvent, fmt.Sprintf("DTEND;VALUE=DATE:%s", dtend))
		} else {
			// Add TZID for timed events if missing
			fixedEvent = append(fixedEvent, fmt.Sprintf("DTEND;TZID=%s:%s", timezone, dtend))
		}
	}
	
	// Add RRULE if present
	if rrule != "" {
		fixedEvent = append(fixedEvent, fmt.Sprintf("RRULE:%s", rrule))
	}
	
	// Add remaining properties
	if location != "" {
		fixedEvent = append(fixedEvent, fmt.Sprintf("LOCATION:%s", location))
	}
	
	if description != "" {
		fixedEvent = append(fixedEvent, fmt.Sprintf("DESCRIPTION:%s", description))
	}
	
	// Add other properties
	fixedEvent = append(fixedEvent, otherProps...)
	
	fixedEvent = append(fixedEvent, "END:VEVENT")
	return fixedEvent
}