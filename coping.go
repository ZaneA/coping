// Coping

package main

import (
	"flag"
	"log"
	"time"
	"net/http"
	"io"
	"io/ioutil"
	"encoding/json"
	"strings"
)

// Stores the result of a "ping"
type FetchResult struct {
	url string
	code int
	requestTime time.Duration
}

// Return whether the status is a pass or fail
func (w FetchResult) Status() bool {
	if w.requestTime > (1 * time.Second) {
		return false
	}

	if w.code != 200 {
		return false
	}

	return true
}

// Convert a status into a PASS/WARN/FAIL string
func (w FetchResult) StatusString() string {
	if w.Status() == true {
		return "\x1b[1;32mPASS\x1b[0m"
	} else {
		if (w.code == -1) {
			return "\x1b[1;31mFAIL\x1b[0m"
		} else {
			return "\x1b[0;33mWARN\x1b[0m"
		}
	}
}

// Ping a service
func PingService(url string, report chan FetchResult) {
	start := time.Now()
	res, err := http.Get(url)

	requestTime := time.Since(start)

	if (err != nil) {
		report <- FetchResult{url, -1, requestTime}
		return
	}

	report <- FetchResult{url, res.StatusCode, requestTime}
}

// Hold services
type ServicesResult struct {
	buddy string
	services []string
}

// Fetch services
func FetchServices(buddy string, report chan ServicesResult) {
	res, err := http.Get(buddy + "/services")

	if err != nil {
		log.Printf("\x1b[1;31m Buddy stopped responding ... %s\x1b[0m\n", buddy)
		return
	}

	defer res.Body.Close()

	result := ServicesResult{buddy,nil}

	body, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &result.services)

	report <- result
}

// GET /services
func WebServicesHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(settings.Services)
	io.WriteString(w, string(b))
}

// Hold buddies
type BuddiesResult struct {
	buddy string
	buddies []string
}

// Fetch buddies
func FetchBuddies(buddy string, report chan BuddiesResult) {
	res, err := http.Get(buddy + "/buddies?callback=http://127.0.0.1:" + settings.Port)

	if err != nil {
		return
	}

	defer res.Body.Close()

	result := BuddiesResult{buddy,nil}

	body, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &result.buddies)

	report <- result
}

// GET /buddies
func WebBuddiesHandler(w http.ResponseWriter, r *http.Request) {
	buddy := r.FormValue("callback")

	// If there is a callback, add it to the buddy list
	if buddy != "" {
		found := false

		for _, s := range settings.Buddies {
			if s == buddy {
				found = true
				break
			}
		}

		if !found {
			settings.Buddies = append(settings.Buddies, buddy)
			log.Printf("\x1b[1;32m Got new buddy from request ... %s\x1b[0m\n", buddy)
		}
	}

	b, _ := json.Marshal(settings.Buddies)
	io.WriteString(w, string(b))
}

// POST /report
func WebReportHandler(w http.ResponseWriter, r *http.Request) {
}

type Settings struct {
	Port string
	Buddies []string
	Services []string
}

var settings = Settings{}

func LoadSettings(file string) {
	body, err := ioutil.ReadFile(file)

	if err != nil {
		log.Fatalf("Couldn't read %s!\n", file)
	}

	err = json.Unmarshal(body, &settings);

	if err != nil {
		log.Fatalf("Couldn't parse %s: %s\n", file, err.Error())
	}
}

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
}

func main() {
	LoadSettings("config.json")

	port := flag.String("port", "9999", "Port to listen on")
	buddies := flag.String("buddies", "", "Comma-separated list of buddies to use for bootstrapping")
	services := flag.String("services", "", "Comma-separated list of services to check")

	flag.Parse()

	if (*buddies != "") {
		settings.Buddies = strings.Split(*buddies, ",")
	}

	if (*services != "") {
		settings.Services = strings.Split(*services, ",")
	}

	settings.Port = *port

	// Start webserver
	http.HandleFunc("/services", WebServicesHandler) // Return a list of services for sharing with other instances of coping
	http.HandleFunc("/buddies", WebBuddiesHandler) // Return a list of buddies for sharing with other instances of coping
	http.HandleFunc("/report", WebReportHandler) // ????
	go http.ListenAndServe(":" + *port, nil)

	log.Printf("\x1b[1;33mCoping is listening on http://127.0.0.1:" + *port + "\x1b[0m\n")

	// Set up fetch tick
	checkTicker := time.Tick(10 * time.Second)
	serviceListTicker := time.Tick(15 * time.Second)
	buddyListTicker := time.Tick(30 * time.Second)

	fetchResultChan := make(chan FetchResult)
	servicesResultChan := make(chan ServicesResult)
	buddiesResultChan := make(chan BuddiesResult)

	buddyServices := map[string][]string{}

	// Loop
	for {
		select {
		case <- checkTicker:
			for _, s := range settings.Services {
				go PingService(s, fetchResultChan)
			}

		case result := <- fetchResultChan:
			log.Printf("[%s] %s (status %d) fetched in %s\n", result.StatusString(), result.url, result.code, result.requestTime.String())

		case <- serviceListTicker:
			for _, buddy := range settings.Buddies {
				go FetchServices(buddy, servicesResultChan)
			}

		case <- buddyListTicker:
			for _, buddy := range settings.Buddies {
				go FetchBuddies(buddy, buddiesResultChan)
			}

		case result := <- servicesResultChan:
			for _, service := range result.services {
				found := false

				for _, s := range settings.Services {
					if s == service {
						found = true
						break
					}
				}

				if !found {
					settings.Services = append(settings.Services, service)
					log.Printf("\x1b[1;32mGot new service from %s ... %s\x1b[0m\n", result.buddy, service)
				}
			}
			buddyServices[result.buddy] = result.services

		case result := <- buddiesResultChan:
			for _, buddy := range result.buddies {
				found := false

				for _, s := range settings.Buddies {
					if s == buddy {
						found = true
						break
					}
				}

				if !found {
					settings.Buddies = append(settings.Buddies, buddy)
					log.Printf("\x1b[1;32mGot new buddy from %s ... %s\x1b[0m\n", result.buddy, buddy)
				}
			}
		}
	}
}
