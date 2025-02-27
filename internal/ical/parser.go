package ical

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/arran4/golang-ical"
)

// FetchCalendar retrieves an iCalendar from a URL or local file path
func FetchCalendar(source string) (*ics.Calendar, error) {
	var calData []byte
	var err error

	// Check if the source is a local file
	if strings.HasPrefix(source, "file://") {
		// Extract the file path from the URL
		filePath := strings.TrimPrefix(source, "file://")
		calData, err = os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
	} else {
		// Treat as a URL
		resp, err := http.Get(source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		calData, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	cal, err := ics.ParseCalendar(strings.NewReader(string(calData)))
	if err != nil {
		return nil, err
	}

	return cal, nil
}

// Event represents a calendar event with additional metadata
type Event struct {
	UID         string
	Summary     string
	CalendarIDs []string
	OriginalEvent *ics.VEvent
}

// MergeCalendars combines multiple calendars into one, handling duplicates
func MergeCalendars(calendars map[string]*ics.Calendar) *ics.Calendar {
	merged := ics.NewCalendar()
	merged.SetMethod(ics.MethodPublish)
	merged.SetProductId("-//ical_merger//GO")
	
	// Track events by summary to identify duplicates
	eventMap := make(map[string]*Event)
	
	// First pass: identify duplicates
	for calID, cal := range calendars {
		for _, event := range cal.Events() {
			summary := event.GetProperty(ics.ComponentPropertySummary).Value
			uid := event.GetProperty(ics.ComponentPropertyUniqueId).Value
			
			if existing, ok := eventMap[summary]; ok {
				// This is a duplicate event, add calendar ID to the list
				existing.CalendarIDs = append(existing.CalendarIDs, calID)
			} else {
				// New event
				eventMap[summary] = &Event{
					UID:           uid,
					Summary:       summary,
					CalendarIDs:   []string{calID},
					OriginalEvent: event,
				}
			}
		}
	}
	
	// Second pass: add events to merged calendar with modified summaries if needed
	for _, event := range eventMap {
		// Create a new event with the same UID
		newEvent := ics.NewEvent(event.UID)
		
		// Copy key properties from original event
		if summary := event.OriginalEvent.GetProperty(ics.ComponentPropertySummary); summary != nil {
			newEvent.SetProperty(ics.ComponentPropertySummary, summary.Value)
		}
		if dtstart := event.OriginalEvent.GetProperty(ics.ComponentPropertyDtStart); dtstart != nil {
			newEvent.SetProperty(ics.ComponentPropertyDtStart, dtstart.Value)
		}
		if dtend := event.OriginalEvent.GetProperty(ics.ComponentPropertyDtEnd); dtend != nil {
			newEvent.SetProperty(ics.ComponentPropertyDtEnd, dtend.Value)
		}
		if loc := event.OriginalEvent.GetProperty(ics.ComponentPropertyLocation); loc != nil {
			newEvent.SetProperty(ics.ComponentPropertyLocation, loc.Value)
		}
		if desc := event.OriginalEvent.GetProperty(ics.ComponentPropertyDescription); desc != nil {
			newEvent.SetProperty(ics.ComponentPropertyDescription, desc.Value)
		}
		
		// If the event appears in only one calendar, prepend the calendar name
		if len(event.CalendarIDs) == 1 {
			summaryProp := newEvent.GetProperty(ics.ComponentPropertySummary)
			if summaryProp != nil {
				originalSummary := summaryProp.Value
				newSummary := "[" + event.CalendarIDs[0] + "] " + originalSummary
				
				// Update the summary with SetProperty
				newEvent.SetProperty(ics.ComponentPropertySummary, newSummary)
			}
		}
		
		merged.AddVEvent(newEvent)
	}
	
	return merged
}