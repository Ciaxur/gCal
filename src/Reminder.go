package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"google.golang.org/api/calendar/v3"
)

// Reminder Structure
type Reminder struct {
	didRemind bool  // If Reminder was Executed
	minBefore int64 // Minutes Prior to Event
}

// Event Structure
type Event struct {
	// Event Basic Info
	id      string
	sumamry string

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
		id:            eList[eID].id,
		sumamry:       eList[eID].sumamry,
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
			eList[item.Id] = Event{
				id:            item.Id,
				sumamry:       item.Summary,
				reminders:     make([]Reminder, 0), // Reset Reminders
				startDate:     item.Start.Date,
				startDateTime: item.Start.DateTime,
				endDate:       item.End.Date,
				endDateTime:   item.End.DateTime,
			}
		}
	}
}

// Parses Given String as RFC3339 Date
func parseDate(date string) time.Time {
	btrDate, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Fatalf("Data Parse [%v] Failed: %v", date, err)
	}
	return btrDate
}

/** Validates to see whether to remind Event
 *  and stores it's ID to keep track of it
 * @param eList A Map of the Event IDs that were Notified
 * @param item Pointer to the Calendar Event
 */
func checkRemind(eList map[string]Event, item *calendar.Event) {
	// ID Should'nt be Used before
	//  or wasn't Reminded Before
	val, ok := eList[item.Id]

	// Check if Tracked
	if !ok { // Keep Track of Event
		eList[item.Id] = Event{
			item.Id,
			item.Summary,
			make([]Reminder, 0),
			item.Start.Date,
			item.End.Date,
			item.Start.DateTime,
			item.End.DateTime,
		}
	} else { // Check for Reminders
		// Get Current Time
		nowTime := time.Now()

		// Parse Event's Start Time
		eDateStr := item.Start.DateTime
		if len(eDateStr) == 0 {
			eDateStr = item.Start.Date + "T00:00:00-04:00"
		}
		eventTime := parseDate(eDateStr)

		// Time till Event
		dEventTime := eventTime.Sub(nowTime)

		for i, reminder := range val.reminders {
			// Check when to Remind
			remIn := math.Floor(dEventTime.Minutes() - float64(reminder.minBefore))

			// Threshold of 0-1min
			if !reminder.didRemind && (remIn <= 0.0 && remIn >= -1.0) {
				fmt.Printf("Reminder: [%v](In %.0f) \n%v\n", item.Summary, dEventTime.Minutes(), item.Summary)
				notifySend(item.Summary, item.Description, int64(dEventTime.Minutes()))
				val.reminders[i].didRemind = true
			}

		}
	}

}

/** Prints out Event to stdout in a Neat Way
  * @param title The Title of the Event
	* @param event Pointer to the Event to print
*/
func printEvent(event Event) {
	// Basic Event Information
	Out.Info.Printf("[%s]\n", event.sumamry)
	fmt.Printf("\t - ID: %s\n", event.id)
	fmt.Printf("\t - Date Range: %s - %s\n", event.startDate, event.endDate)
	fmt.Printf("\t - DateTime Range: %s - %s\n", event.startDateTime, event.endDateTime)

	// Reminder Information
	fmt.Printf("\t - Reminders: \n")
	for i, reminder := range event.reminders {
		fmt.Printf("\t\t - Reminder[%d]\n", i)
		fmt.Printf("\t\t\t - Minutes Before: %dmin\n", reminder.minBefore)
		fmt.Printf("\t\t\t - Did Remind: %v\n", reminder.didRemind)

		// Time Till Reminder (Only Non-Reminded)
		if !reminder.didRemind && len(event.id) != 0 { // Make sure Event Data is Stored
			date := event.startDateTime
			if len(date) == 0 {
				date = event.startDate + "T00:00:00-04:00"
			}
			rTime := parseDate(date)
			tRemind := rTime.Sub(time.Now()).Minutes() - float64(reminder.minBefore)
			fmt.Printf("\t\t\t - Till Reminder: %.1fmin\n", tRemind)
		}
	}
}

/** Garbage Collection for Events
 * Looks through to see if an Event Passed
 *  a threshold of time
 * @param eList Map of Events
 */
func cleanupEvents(eList map[string]Event) {
	// Keep Track of Current Time and Event Date
	now := time.Now()
	eventDate := ""

	for key, val := range eList {
		if len(val.endDate) != 0 { // Entire Day Event
			eventDate = val.endDate + "T00:00:00-04:00"
		} else if len(val.endDateTime) != 0 { // Date Time Event
			eventDate = val.endDateTime
		}

		// Garbage Collection
		// Try to Parse Date
		d, err := time.Parse(time.RFC3339, eventDate)
		if err != nil {
			log.Printf("Garbage Collection: Error parsing %s\n", eventDate)
		} else {
			if d.Sub(now).Hours() < -2.00 { // Remove if 2 Hours Past
				delete(eList, key)
				continue
			}
		}

	}
}
