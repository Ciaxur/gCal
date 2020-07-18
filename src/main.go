package main

import (
	"flag"
	"io/ioutil"
	"log"
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
	isVerbose bool // Verbose Mode for Debuggin Purposes
	apiRate   int  // Override Google Calendar API Request Rate
}

// Parse through the CLI Arguments
// Returning Flags
func parseInput() cliArguments {
	var eventNum = flag.Int("events", 10, "Number of Events Accounted for")
	flag.IntVar(eventNum, "e", 10, "Number of Events Accounted for")

	var isVerbose = flag.Bool("verbose", false, "Enable Verbose Mode for Debug Prints")

	var apiRate = flag.Int("rate", 30, "Google Calendar API Request Rate")

	flag.Parse()

	return cliArguments{*eventNum, *isVerbose, *apiRate}
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

		if args.isVerbose {
			Out.Info.Println("Upcoming Events: ")
		}
		if len(events.Items) == 0 {
			Out.Warning.Println("No Upcoming events found!")
		} else {
			// Indicate Header for Verbose
			if args.isVerbose {
				Out.Info.Print("\n== Reminder Info ==\n")
			}

			// Handle Each Event
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

				// Integrity Check: Any Modifications to Events
				checkIntegrity(item, eventsMap)

				// New Reminders: Add New Reminders to Map
				addReminders(reminders, item.Id, eventsMap)

				// Notify: Check Reminders
				checkRemind(eventsMap, item)

				// Verbose Mode
				if args.isVerbose {
					printEvent(eventsMap[item.Id])
				}
			}
		}

		// VERBOSE: Better Output
		if args.isVerbose {
			println()
		}

		// Sleep for 30 Seconds
		time.Sleep(time.Duration(args.apiRate) * time.Second)

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
