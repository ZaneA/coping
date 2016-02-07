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

func Contains(needle string, haystack *[]string) bool {
	for _, v := range *haystack {
		if v == needle {
			return true
		}
	}

	return false
}

type Settings struct {
	Port             int
	AlertCount       int
	CheckInterval    int
	ServicesInterval int
	BuddiesInterval  int
	Buddies          []string
	Services         []string
}

func (settings Settings) GetCallback() string {
	return "http://127.0.0.1:" + strconv.Itoa(settings.Port)
}

// Hold services
type ServicesResult struct {
	Buddy    string
	Services []string
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
	json.Unmarshal(body, &result.Services)

	report <- result
}

// Hold buddies
type BuddiesResult struct {
	Buddy   string
	Buddies []string
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
	json.Unmarshal(body, &result.Buddies)

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

// GET /buddies
func WebBuddiesHandler(w http.ResponseWriter, r *http.Request) {
	CheckForCallback(r)

	b, _ := json.Marshal(settings.Buddies)
	io.WriteString(w, string(b))
}

// GET /services
func WebServicesHandler(w http.ResponseWriter, r *http.Request) {
	CheckForCallback(r)

	b, _ := json.Marshal(settings.Services)
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
	flag.IntVar(&settings.ServicesInterval, "servicesInterval", 60, "How often to update services list (in seconds)")
	flag.IntVar(&settings.BuddiesInterval, "buddiesInterval", 120, "How often to update buddies list (in seconds)")

	flag.Parse()

	if *buddies != "" {
		settings.Buddies = strings.Split(*buddies, ",")
	}

	if *services != "" {
		settings.Services = strings.Split(*services, ",")
	}

	// Start webserver
	http.HandleFunc("/services", WebServicesHandler)
	http.HandleFunc("/buddies", WebBuddiesHandler)
	go http.ListenAndServe(":"+strconv.Itoa(settings.Port), nil)

	log.Printf("\x1b[1;33mCoping is listening on %s\x1b[0m\n", settings.GetCallback())

	// Set up fetch tick
	checkTicker := time.Tick(time.Duration(settings.CheckInterval) * time.Second)
	serviceListTicker := time.Tick(time.Duration(settings.ServicesInterval) * time.Second)
	buddyListTicker := time.Tick(time.Duration(settings.BuddiesInterval) * time.Second)

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

		case <-serviceListTicker:
			for _, buddy := range settings.Buddies {
				go FetchServices(buddy, servicesResultChan)
			}

		case <-buddyListTicker:
			for _, buddy := range settings.Buddies {
				go FetchBuddies(buddy, buddiesResultChan)
			}

		case result := <-fetchResultChan:
			_, status := result.StatusString()
			log.Printf("[%s] %s (status %d) fetched in %v\n", status, result.Url, result.Code, result.Duration)
			go MaybeAlert(&settings, result)

		case result := <-servicesResultChan:
			for _, service := range result.Services {
				if !Contains(service, &settings.Services) {
					settings.Services = append(settings.Services, service)
					log.Printf("\x1b[1;32mGot new service from %s ... %s\x1b[0m\n", result.Buddy, service)
				}
			}

		case result := <-buddiesResultChan:
			for _, buddy := range result.Buddies {
				if !Contains(buddy, &settings.Buddies) {
					settings.Buddies = append(settings.Buddies, buddy)
					log.Printf("\x1b[1;32mGot new buddy from %s ... %s\x1b[0m\n", result.Buddy, buddy)
				}
			}
		}
	}
}
