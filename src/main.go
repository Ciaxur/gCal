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

	// Setup Title
	title := "Google Calendar (" + strconv.FormatInt(eventDiff, 10) + "min Reminder)"

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

	for {
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
				}

				for i, d := range reminders {
					remIn := math.Floor(eventDiff.Minutes() - float64(d))

					// List only
					if args.isList {
						fmt.Printf("\t -Reminder [%d] = %dmin \t Remind in [%.2fmin]\n", i, d, remIn)
					} else {
						// Check to Remind!
						if remIn == 0.0 { // Issue a Reminder
							notifySend(item.Summary, item.Description, int64(remIn))
						}

					}
				}

			}
		}

		// Sleep for 10 Seconds
		time.Sleep(10 * time.Second)
	}

}
