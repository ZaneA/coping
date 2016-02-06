// Coping

package main

import (
	"log"
	"time"
	"net/http"
	"io"
	"io/ioutil"
	"encoding/json"
)

type FetchResult struct {
	buddy string
	url string
	code int
	requestTime time.Duration
}

func (w FetchResult) Status() bool {
	if w.requestTime > (1 * time.Second) {
		return false
	}

	if w.code != 200 {
		return false
	}

	return true
}

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

func PingService(buddy string, url string, report chan FetchResult) {
	start := time.Now()
	res, err := http.Get(url)

	requestTime := time.Since(start)

	if (err != nil) {
		report <- FetchResult{buddy, url, -1, requestTime}
		return
	}

	report <- FetchResult{buddy, url, res.StatusCode, requestTime}
}

type ServicesResult struct {
	buddy string
	services []string
}

func FetchServices(buddy string, report chan ServicesResult) {
	res, _ := http.Get(buddy + "/services")
	result := ServicesResult{buddy,nil}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	json.Unmarshal(body, &result.services)
	report <- result
}

// GET /services
func WebServicesHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(settings.Services)
	io.WriteString(w, string(b))
}

// POST /report
func WebReportHandler(w http.ResponseWriter, r *http.Request) {
}

type Settings struct {
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

	// Start webserver
	http.HandleFunc("/services", WebServicesHandler)
	http.HandleFunc("/report", WebReportHandler)
	go http.ListenAndServe(":9999", nil)

	log.Printf("[\x1b[1;33mSTATUS\x1b[0m] Coping is listening on http://127.0.0.1:9999\n")

	// Set up fetch tick
	checkTicker := time.Tick(10 * time.Second)
	serviceListTicker := time.Tick(15 * time.Second)

	fetchResultChan := make(chan FetchResult)
	servicesResultChan := make(chan ServicesResult)

	buddyServices := map[string][]string{}

	// Loop
	for {
		select {
		case <- checkTicker:
			for b, s := range buddyServices {
				for _, v := range s {
					go PingService(b, v, fetchResultChan)
				}
			}

		case result := <- fetchResultChan:
			log.Printf("[%s] %s (status %d) fetched in %s\n", result.StatusString(), result.url, result.code, result.requestTime.String())

		case <- serviceListTicker:
			log.Printf("[\x1b[1;33mSTATUS\x1b[0m] Updating list of services from buddies...\n")
			for _, buddy := range settings.Buddies {
				go FetchServices(buddy, servicesResultChan)
			}

		case result := <- servicesResultChan:
			log.Printf("[\x1b[1;33mSTATUS\x1b[0m] Got services from %s:\n", result.buddy)
			for _, service := range result.services {
				log.Printf("[\x1b[1;33mSTATUS\x1b[0m] ... %s\n", service)
			}
			buddyServices[result.buddy] = result.services
		}
	}
}
