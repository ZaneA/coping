// Coping

package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	Port          int
	AlertCount    int
	CheckInterval int
	SyncInterval  int
	Buddies       []string
	Services      []string
}

func (settings Settings) GetCallback() string {
	return "http://127.0.0.1:" + strconv.Itoa(settings.Port)
}

// Hold services
type SyncJson struct {
	Services []string
	Buddies  []string
}

type SyncResult struct {
	Buddy string
	Data  SyncJson
}

// Fetch services
func Sync(buddy string, report chan SyncResult) {
	res, err := http.Get(buddy + "/sync?callback=" + settings.GetCallback())

	if err != nil {
		log.Printf("\x1b[1;31m Buddy not responding to /sync ... %s\x1b[0m\n", buddy)
		return
	}

	defer res.Body.Close()

	result := SyncResult{buddy, SyncJson{nil, nil}}

	body, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &result.Data)

	report <- result
}

func CheckForCallback(r *http.Request) {
	buddy := r.FormValue("callback")

	// If there is a callback, add it to the buddy list
	if buddy != "" {
		if !Contains(buddy, &settings.Buddies) {
			settings.Buddies = append(settings.Buddies, buddy)
			log.Printf("\x1b[1;32mGot new buddy from request ... %s\x1b[0m\n", buddy)
		}
	}
}

// GET /sync
func WebSyncHandler(w http.ResponseWriter, r *http.Request) {
	CheckForCallback(r)

	state := SyncJson{settings.Services, settings.Buddies}

	b, _ := json.Marshal(state)
	io.WriteString(w, string(b))
}

// Global
var settings = Settings{}

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.SetOutput(os.Stderr)
}

func main() {
	flag.IntVar(&settings.Port, "port", 9999, "Port to listen on")
	flag.IntVar(&settings.AlertCount, "alertCount", 3, "How many times a service can report failure before alerting")

	buddies := flag.String("buddies", "", "Comma-separated list of buddies to use for bootstrapping")
	services := flag.String("services", "", "Comma-separated list of services to check")

	flag.IntVar(&settings.CheckInterval, "checkInterval", 60, "How often to check services (in seconds)")
	flag.IntVar(&settings.SyncInterval, "syncInterval", 60, "How often to sync services/buddies state (in seconds)")

	flag.Parse()

	if *buddies != "" {
		settings.Buddies = strings.Split(*buddies, ",")
	}

	if *services != "" {
		settings.Services = strings.Split(*services, ",")
	}

	// Start webserver
	http.HandleFunc("/sync", WebSyncHandler)
	go http.ListenAndServe(":"+strconv.Itoa(settings.Port), nil)

	log.Printf("\x1b[1;33mCoping is listening on %s\x1b[0m\n", settings.GetCallback())

	// Set up fetch tick
	checkTicker := time.Tick(time.Duration(settings.CheckInterval) * time.Second)
	syncTicker := time.Tick(time.Duration(settings.SyncInterval) * time.Second)

	checkResultChan := make(chan CheckResult)
	syncResultChan := make(chan SyncResult)

	// Loop
	for {
		select {
		case <-checkTicker:
			for _, s := range settings.Services {
				go CheckService(s, checkResultChan)
			}

		case <-syncTicker:
			for _, buddy := range settings.Buddies {
				go Sync(buddy, syncResultChan)
			}

		case result := <-checkResultChan:
			_, status := result.StatusString()
			log.Printf("[%s] %s (status %d) fetched in %v\n", status, result.Url, result.Code, result.Duration)
			go MaybeAlert(&settings, result)

		case result := <-syncResultChan:
			for _, buddy := range result.Data.Buddies {
				if !Contains(buddy, &settings.Buddies) {
					settings.Buddies = append(settings.Buddies, buddy)
					log.Printf("\x1b[1;32mGot new buddy from %s ... %s\x1b[0m\n", result.Buddy, buddy)
				}
			}

			for _, service := range result.Data.Services {
				if !Contains(service, &settings.Services) {
					settings.Services = append(settings.Services, service)
					log.Printf("\x1b[1;32mGot new service from %s ... %s\x1b[0m\n", result.Buddy, service)
				}
			}
		}
	}
}
