// Coping

package main

import (
	"fmt"
	"time"
	"net/http"
	"io"
	"io/ioutil"
	"encoding/json"
)

type FetchResult struct {
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
		return "PASS"
	} else {
		return "FAIL"
	}
}

func PingService(url string, report chan FetchResult) {
	start := time.Now()
	res, _ := http.Get(url)
	requestTime := time.Since(start)
	
	report <- FetchResult{url, res.StatusCode, requestTime}
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
	json.Unmarshal(body, result.services)
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

var settings = Settings{
	Buddies: []string{
		"http://127.0.0.1:9999",
	},
	Services: []string{
		"http://127.0.0.1/coping/pass.php",
		"http://127.0.0.1/coping/slow.php",
		"http://127.0.0.1/coping/error.php",
	},
}

func main() {
	// Start webserver
	http.HandleFunc("/services", WebServicesHandler)
	http.HandleFunc("/report", WebReportHandler)
	go http.ListenAndServe(":9999", nil)
	
	fmt.Printf("Coping is now listening on http://127.0.0.1:9999\n")

	// Set up fetch tick
	checkTicker := time.Tick(10 * time.Second)
	serviceListTicker := time.Tick(15 * time.Second)
	
	fetchResultChan := make(chan FetchResult)
	servicesResultChan := make(chan ServicesResult)

	// Loop
	for {
		select {
		case <- checkTicker:
			fmt.Println("Heartbeat")
			for i := 0; i < len(settings.Services); i++ {
				go PingService(settings.Services[i], fetchResultChan)
			}
			
		case result := <- fetchResultChan:
			fmt.Printf("%s was fetched with status code %d in %s [%s]\n", result.url, result.code, result.requestTime.String(), result.StatusString())
		
		case <- serviceListTicker:
			fmt.Println("Fetching list of services from buddies...")
			for i := 0; i < len(settings.Buddies); i++ {
				go FetchServices(settings.Buddies[i], servicesResultChan)
			}
			
		case result := <- servicesResultChan:
			fmt.Printf("Got services from %s...", result.buddy)
		}
	}
}
