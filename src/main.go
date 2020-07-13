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
	eventNum  int  // Number of Events to Look for (Default = 10)
	isList    bool // Only List Events
	stillRun  bool // Runs Notification even if only List enabled (Default = False)
	isVerbose bool // Verbose Mode for Debuggin Purposes
}

// Parse through the CLI Arguments
// Returning Flags
func parseInput() cliArguments {
	var flagList = flag.Bool("list", false, "Only List the Events")
	flag.BoolVar(flagList, "l", false, "Only List the Events")

	var eventNum = flag.Int("events", 10, "Number of Events Accounted for")
	flag.IntVar(eventNum, "e", 10, "Number of Events Accounted for")

	var stilLRun = flag.Bool("run", false, "Still Run even if only List Enabled")
	flag.BoolVar(stilLRun, "r", false, "Still Run even if only List Enabled")

	var isVerbose = flag.Bool("verbose", false, "Enable Verbose Mode for Debug Prints")

	flag.Parse()

	return cliArguments{*eventNum, *flagList, *stilLRun, *isVerbose}
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

	// Variables Used
	eventsMap := map[string]Event{} // Calendar Event List
	iSinceCleanup := 0              // Keep Track of Iterations from Cleanup
	cleanupFrequency := 20          // How Often to Issue a Clean up of Events

	for { // Keep Watching
		t := time.Now().Format(time.RFC3339)

		// Obtain Recent Events
		events, err := srv.Events.List("primary").ShowDeleted(false).
			SingleEvents(true).TimeMin(t).MaxResults(int64(args.eventNum)).OrderBy("startTime").Do()
		if err != nil {
			log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
		}

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
						fmt.Printf("[%v] (%d/%d/%d)\n", item.Summary, btrDate.Day(), btrDate.Month(), btrDate.Year())
					} else {
						fmt.Printf("[%v] (%v)\n", item.Summary, btrDate.Format(time.Stamp))
					}

					// Print the Difference till Event
					fmt.Printf("\t -Time Till Event: %.2fmin\n", eventDiff.Minutes())
					// Print out the IDs
					fmt.Printf("\t -ID: %v\n", item.Id)
				}

				// Go through each Event's Reminder
				for i, d := range reminders {
					remIn := math.Floor(eventDiff.Minutes() - float64(d))

					// List only
					if args.isList {
						fmt.Printf("\t -Reminder [%d] = %dmin \t Remind in [%.2fmin]\n", i, d, remIn)
					}

					// Check Reminders if List or not
					if !args.isList || (args.isList && args.stillRun) {
						// Check Integrity for Changes
						checkIntegrity(item, eventsMap)

						// Check to Remind!
						checkRemind(eventsMap, item, remIn, int64(eventDiff.Minutes()))

						// Verbose Mode
						if args.isVerbose {
							printEvent(eventsMap[item.Id])
						}
					}
				}

				// Add New Reminders to Map
				addReminders(reminders, item.Id, eventsMap)

			}
		}

		// Break out if just Listing
		if args.isList && !args.stillRun {
			os.Exit(0)
		}

		// Better Output
		if args.isList {
			println()
		}

		// Sleep for 30 Seconds
		time.Sleep(30 * time.Second)

		// Increment Iteration
		iSinceCleanup++

		// Check Garbage Collection
		if iSinceCleanup >= cleanupFrequency {
			iSinceCleanup = 0        // Reset Cleanup
			cleanupEvents(eventsMap) // Issue Clean up

			// Log Cleanup
			if args.isVerbose {
				log.Printf("Garbage Collection Issued\n")
			}
		}

	}

}
