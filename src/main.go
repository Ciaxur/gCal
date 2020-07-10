package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"google.golang.org/api/calendar/v3"

	"golang.org/x/oauth2/google"
)

// Structure for Valid CLI Arguments
type cliArguments struct {
	eventNum int  // Number of Events to Look for (Default = 10)
	isList   bool // Only List Events
}

// Parse through the CLI Arguments
// Returning Flags
func parseInput() cliArguments {
	var flagList = flag.Bool("List", false, "Only List the Events")
	flag.BoolVar(flagList, "l", false, "Only List the Events")

	var eventNum = flag.Int("Events", 10, "Number of Events Accounted for")
	flag.IntVar(eventNum, "e", 10, "Number of Events Accounted for")

	flag.Parse()

	return cliArguments{*eventNum, *flagList}
}

// Wrapper around notify-send
func notifySend(summary string, description string, eventDiff int64) {
	// Obtain Current Path
	binPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	// Setup Title | in or ago
	title := "Google Calendar (in " + strconv.FormatInt(eventDiff, 10) + "min)"
	if eventDiff < 0 {
		title = "Google Calendar (" + strconv.FormatInt(eventDiff*-1, 10) + "min ago)"
	}

	// INITIATE NOTIFICATION
	cmd := exec.Command(
		"notify-send",
		summary, description,
		"-i", binPath+"/calendar.png",
		"-u", "normal",
		"-a", title)
	cmd.Start()
}

func main() {
	// Parse Arugments
	args := parseInput()

	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// Congiure Credentials to JSON Object
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse Client Secret File to Config: %v", err)
	}

	// Create a Client and a Calendar Service Object
	client := getClient(config)
	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrienve Calendar Client: %v", err)
	}

	// Check every minute
	// for {
	t := time.Now().Format(time.RFC3339)

	// Obtain Recent Events
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(int64(args.eventNum)).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	// Variables Used
	eventsDone := map[string]Event{}

	for { // Keep Watching
		// Check Events
		nowTime := time.Now()

		if args.isList {
			fmt.Println("Upcoming Events: ")
		}
		if len(events.Items) == 0 {
			fmt.Println("No Upcoming events found!")
		} else {

			for _, item := range events.Items {
				// Get Reminder Times [minutes prior to event]
				reminders := make([]int64, 0, 4)

				if item.Reminders.UseDefault {
					reminders = append(reminders, 10)
				} else {
					for _, rem := range item.Reminders.Overrides {
						reminders = append(reminders, rem.Minutes)
					}
				}

				// Specific Time Range
				date := item.Start.DateTime

				// Parse Date
				var btrDate time.Time
				isEntireDay := false

				// Parse Date
				if date == "" {
					isEntireDay = true
					date = item.Start.Date + "T00:00:00-04:00"
					btrDate, err = time.Parse(time.RFC3339, date)
				} else {
					btrDate, err = time.Parse(time.RFC3339, date)
				}
				if err != nil {
					log.Fatalf("Data Parse [%v] Failed: %v", date, err)
				}

				// Time till Event
				eventDiff := btrDate.Sub(nowTime)

				// List Only
				if args.isList {
					if isEntireDay {
						fmt.Printf("%v (%d/%d/%d)\n", item.Summary, btrDate.Day(), btrDate.Month(), btrDate.Year())
					} else {
						fmt.Printf("%v (%v)\n", item.Summary, btrDate.Format(time.Stamp))
					}

					// Print the Difference till Event
					fmt.Printf("\t -Time Till Event: %.2fmin\n", eventDiff.Minutes())
					// Print out the IDs
					fmt.Printf("\t -ID: %v\n", item.Id)
				}

				for i, d := range reminders {
					remIn := math.Floor(eventDiff.Minutes() - float64(d))

					// List only
					if args.isList {
						fmt.Printf("\t -Reminder [%d] = %dmin \t Remind in [%.2fmin]\n", i, d, remIn)
					} else {
						// Check Integrity for Changes
						checkIntegrity(item, eventsDone)

						// Check to Remind!
						checkRemind(eventsDone, item, remIn, int64(eventDiff.Minutes()))
					}
				}

			}
		}

		// Break out if just Listing
		if args.isList {
			os.Exit(0)
		}

		// Sleep for 10 Seconds
		time.Sleep(30 * time.Second)
	}

}

// Event Structure
type Event struct {
	didRemind     bool   // If Reminder was Executed
	startDate     string // Starting Date
	endDate       string // Ending Date
	startDateTime string // Starting Date Time
	endDateTime   string // Ending Date Time
}

// Checks if the Event was Modified and Updates the Map
func checkIntegrity(item *calendar.Event, eList map[string]Event) {
	// Check if it's Stored
	if val, ok := eList[item.Id]; ok {
		// Check if Values Changed
		if val.startDate != item.Start.Date ||
			val.endDate != item.End.Date ||
			val.startDateTime != item.Start.DateTime ||
			val.endDateTime != item.End.DateTime {

			// Modify Event
			val = Event{
				didRemind:     false,
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
	if (!ok || !val.didRemind) && remIn <= 0.0 {
		fmt.Printf("Reminder: [%v](%v) \n%v\n", item.Summary, eMinutes, item.Summary)
		notifySend(item.Summary, item.Description, eMinutes)

		// Keep Track of Event
		eList[item.Id] = Event{
			true,
			item.Start.Date,
			item.End.Date,
			item.Start.DateTime,
			item.End.DateTime,
		}
	}
}
