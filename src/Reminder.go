package main

import (
	"fmt"

	"google.golang.org/api/calendar/v3"
)

// Reminder Structure
type Reminder struct {
	didRemind bool  // If Reminder was Executed
	minBefore int64 // Minutes Prior to Event
}

// Event Structure
type Event struct {
	// Event Reminders
	reminders []Reminder

	// Event Date & Time
	startDate     string // Starting Date
	endDate       string // Ending Date
	startDateTime string // Starting Date Time
	endDateTime   string // Ending Date Time
}

// Checks if Value "minBefore" is in Reminder Array
func contains(reminders []Reminder, val int64) bool {
	for _, elt := range reminders {
		if elt.minBefore == val {
			return true
		}
	}
	return false
}

// Adds new Reminders to Event List
func addReminders(reminders []int64, eID string, eList map[string]Event) {
	// Get Reminders from Event List
	pReminders := eList[eID].reminders

	// Check and add New Reminders
	for _, mins := range reminders {
		if !contains(pReminders, mins) {
			pReminders = append(pReminders, Reminder{
				didRemind: false,
				minBefore: mins,
			})
		}
	}

	// Update Event List
	eList[eID] = Event{
		reminders:     pReminders,
		startDate:     eList[eID].startDate,
		endDate:       eList[eID].endDate,
		startDateTime: eList[eID].startDateTime,
		endDateTime:   eList[eID].endDateTime,
	}
}

// Checks if the Event was Modified and Updates the Map
func checkIntegrity(item *calendar.Event, eList map[string]Event) {
	// Check if it's Stored
	if val, ok := eList[item.Id]; ok {
		// Check if Event Time Range Values Changed
		if val.startDate != item.Start.Date ||
			val.endDate != item.End.Date ||
			val.startDateTime != item.Start.DateTime ||
			val.endDateTime != item.End.DateTime {

			// Modify Event
			// Without Reminders
			val = Event{
				reminders:     make([]Reminder, 0), // Reset Reminders
				startDate:     item.Start.Date,
				startDateTime: item.Start.DateTime,
				endDate:       item.End.Date,
				endDateTime:   item.End.DateTime,
			}
		}
	}
}

/** Validates to see whether to remind Event
  *  and stores it's ID to keep track of it
  * @param eList A Map of the Event IDs that were Notified
  * @param item Pointer to the Calendar Event
  * @param remIn Time Difference to wait till Reminder should Pop up
	* @param eMinutes Time of the Event
*/
func checkRemind(eList map[string]Event, item *calendar.Event, remIn float64, eMinutes int64) {
	// ID Should'nt be Used before
	//  or wasn't Reminded Before
	val, ok := eList[item.Id]

	// Check if Tracked
	if !ok { // Keep Track of Event
		eList[item.Id] = Event{
			make([]Reminder, 0),
			item.Start.Date,
			item.End.Date,
			item.Start.DateTime,
			item.End.DateTime,
		}
	} else { // Check for Reminders
		for _, reminder := range val.reminders {

			if !reminder.didRemind && (remIn <= 0.0 && remIn >= -1.0) {
				fmt.Printf("Reminder: [%v](%v) \n%v\n", item.Summary, eMinutes, item.Summary)
				notifySend(item.Summary, item.Description, eMinutes)
				reminder.didRemind = true
			}

		}
	}

}
