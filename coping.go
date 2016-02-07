// Coping

package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Stores the result of a "ping"
type FetchResult struct {
	url         string
	code        int
	requestTime time.Duration
}

// Return whether the status is a pass or fail
func (result FetchResult) Passed() bool {
	if result.requestTime > (1 * time.Second) {
		return false
	}

	if result.code != 200 {
		return false
	}

	return true
}

// Convert a status into a PASS/WARN/FAIL string
func (result FetchResult) StatusString() string {
	if result.Passed() == true {
		return "\x1b[1;32mPASS\x1b[0m"
	} else {
		if result.code == -1 {
			return "\x1b[1;31mFAIL\x1b[0m"
		} else {
			return "\x1b[0;33mWARN\x1b[0m"
		}
	}
}

type Settings struct {
	Port     int
	Buddies  []string
	Services []string
}

func (settings Settings) GetCallback() string {
	return "http://127.0.0.1:" + strconv.Itoa(settings.Port)
}

// Check a service
func CheckService(url string, report chan FetchResult) {
	start := time.Now()
	res, err := http.Get(url)

	requestTime := time.Since(start)

	if err != nil {
		report <- FetchResult{url, -1, requestTime}
		return
	}

	report <- FetchResult{url, res.StatusCode, requestTime}
}

// Hold services
type ServicesResult struct {
	buddy    string
	services []string
}

// Fetch services
func FetchServices(buddy string, report chan ServicesResult) {
	res, err := http.Get(buddy + "/services?callback=" + settings.GetCallback())

	if err != nil {
		log.Printf("\x1b[1;31m Buddy not responding ... %s\x1b[0m\n", buddy)
		return
	}

	defer res.Body.Close()

	result := ServicesResult{buddy, nil}

	body, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &result.services)

	report <- result
}

// GET /services
func WebServicesHandler(w http.ResponseWriter, r *http.Request) {
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
			log.Printf("\x1b[1;32mGot new buddy from request ... %s\x1b[0m\n", buddy)
		}
	}

	b, _ := json.Marshal(settings.Services)
	io.WriteString(w, string(b))
}

// Hold buddies
type BuddiesResult struct {
	buddy   string
	buddies []string
}

// Fetch buddies
func FetchBuddies(buddy string, report chan BuddiesResult) {
	res, err := http.Get(buddy + "/buddies?callback=" + settings.GetCallback())

	if err != nil {
		return
	}

	defer res.Body.Close()

	result := BuddiesResult{buddy, nil}

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
			log.Printf("\x1b[1;32mGot new buddy from request ... %s\x1b[0m\n", buddy)
		}
	}

	b, _ := json.Marshal(settings.Buddies)
	io.WriteString(w, string(b))
}

// POST /report
func WebReportHandler(w http.ResponseWriter, r *http.Request) {
}

// Global
var settings = Settings{}

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
}

func main() {
	port := flag.Int("port", 9999, "Port to listen on")
	buddies := flag.String("buddies", "", "Comma-separated list of buddies to use for bootstrapping")
	services := flag.String("services", "", "Comma-separated list of services to check")

	checkInterval := flag.Int("checkInterval", 60, "How often to check services (in seconds)")
	servicesInterval := flag.Int("servicesInterval", 60, "How often to update services list (in seconds)")
	buddiesInterval := flag.Int("buddiesInterval", 120, "How often to update buddies list (in seconds)")

	flag.Parse()

	if *buddies != "" {
		settings.Buddies = strings.Split(*buddies, ",")
	}

	if *services != "" {
		settings.Services = strings.Split(*services, ",")
	}

	settings.Port = int(*port)

	// Start webserver
	http.HandleFunc("/services", WebServicesHandler) // Return a list of services for sharing with other instances of coping
	http.HandleFunc("/buddies", WebBuddiesHandler)   // Return a list of buddies for sharing with other instances of coping
	go http.ListenAndServe(":"+strconv.Itoa(settings.Port), nil)

	log.Printf("\x1b[1;33mCoping is listening on " + settings.GetCallback() + "\x1b[0m\n")

	// Set up fetch tick
	checkTicker := time.Tick(time.Duration(*checkInterval) * time.Second)
	serviceListTicker := time.Tick(time.Duration(*servicesInterval) * time.Second)
	buddyListTicker := time.Tick(time.Duration(*buddiesInterval) * time.Second)

	fetchResultChan := make(chan FetchResult)
	servicesResultChan := make(chan ServicesResult)
	buddiesResultChan := make(chan BuddiesResult)

	// Loop
	for {
		select {
		case <-checkTicker:
			for _, s := range settings.Services {
				go CheckService(s, fetchResultChan)
			}

		case result := <-fetchResultChan:
			log.Printf("[%s] %s (status %d) fetched in %s\n", result.StatusString(), result.url, result.code, result.requestTime.String())

		case <-serviceListTicker:
			for _, buddy := range settings.Buddies {
				go FetchServices(buddy, servicesResultChan)
			}

		case <-buddyListTicker:
			for _, buddy := range settings.Buddies {
				go FetchBuddies(buddy, buddiesResultChan)
			}

		case result := <-servicesResultChan:
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

		case result := <-buddiesResultChan:
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
