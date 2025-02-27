package ical

import (
	"io"
	"net/http"
	"strings"

	"github.com/arran4/golang-ical"
)

// FetchCalendar retrieves an iCalendar from a URL
func FetchCalendar(url string) (*ics.Calendar, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	cal, err := ics.ParseCalendar(strings.NewReader(string(body)))
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
			uid := event.GetProperty(ics.ComponentPropertyUID).Value
			
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
		// Make a copy of the original event
		newEvent := event.OriginalEvent.Clone()
		
		// If the event appears in only one calendar, prepend the calendar name
		if len(event.CalendarIDs) == 1 {
			originalSummary := newEvent.GetProperty(ics.ComponentPropertySummary).Value
			newSummary := "[" + event.CalendarIDs[0] + "] " + originalSummary
			
			// Update the summary property
			summaryProp := newEvent.GetProperty(ics.ComponentPropertySummary)
			if summaryProp != nil {
				summaryProp.Value = newSummary
			}
		}
		
		merged.AddVEvent(newEvent)
	}
	
	return merged
}