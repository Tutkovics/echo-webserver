package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/octago/sflags/gen/gflag"
)

// example: main.go -name Frontend -delay 9 -port 9090 -cpu 90 -memory 900 -endpoint-url /read -endpoint-cpu 99 -endpoint-delay 98 -endpoint-url /index -endpoint-cpu 22 -endpoint-delay 202
type config struct {
	Name           string   `flag:"name" desc:"Server/service name"`
	InitDelay      uint     `flag:"delay" desc:"Delay after start up [ms]"`
	Port           uint     `flag:"port" desc:"Open port to listen"`
	CPUusage       uint     `flag:"cpu" desc:"CPU usage in idle time [mCPU]"`
	MemoryUsage    uint     `flag:"memory" desc:"Memory usage in idle time [kB]"`
	Endpoints      []string `flag:"endpoint-url" desc:"Endpoints to listen"`
	EndpointsCPU   []uint   `flag:"endpoint-cpu" desc:"CPU usage for the endpoints"`
	EndpointsDelay []uint   `flag:"endpoint-delay" desc:"Delay for each endpoint [ms]"`
	EndpointsCall  []string `flag:"endpoint-call" desc:"If the endpoint need to call other service"`
}

func readConfigParameters() *config {
	// Set default parameters
	c := &config{
		Name:        "Service-#ID",
		InitDelay:   0,
		Port:        8080,
		CPUusage:    50,
		MemoryUsage: 64,
		Endpoints: []string{
			"/index",
			"/health",
		},
		EndpointsCPU: []uint{
			1, //200,
			1,
		},
		EndpointsDelay: []uint{
			30,
			0,
		},
		EndpointsCall: []string{
			"",
			"asd__basf",
		},
	}

	err := gflag.ParseToDef(c)
	if err != nil {
		log.Fatalf("[READ_PARAMS]\terr: %v", err)
	}
	flag.Parse()

	// Check given paramters
	fmt.Printf("[READ_PARAMS]\tParameters OK: %t\n", c.check())

	return c
}

func (c *config) check() bool {
	if len(c.Endpoints) == len(c.EndpointsCPU) &&
		len(c.Endpoints) == len(c.EndpointsDelay) &&
		len(c.Endpoints) == len(c.EndpointsCall) {
		return true
	}

	return false
}

type Response struct {
	ServiceName      string        `json:"service"`          // Name of the service
	Host             string        `json:"host"`             // Name of the host who answer
	ConfigOK         bool          `json:"config"`           // Results of parameter check
	CalledEnpoint    string        `json:"endpoint"`         // Name of the called endpoint
	CPU              int32         `json:"cpu"`              // CPU usage for the endpoint
	Delay            int32         `json:"delay"`            // Delay time of the endpoint
	CalloutParameter string        `json:"calloutparameter"` // Commandline parameter given in start
	Callouts         []string      `json:"callouts"`         //[]Response // Responses from callouts
	ActualDelay      time.Duration `json:"actualDelay"`      // Actual delay in response
	Time             time.Time     `json:"time"`             // Current time in response
	RequestMethod    string        `json:"requestMethod"`    // Method of request
	RequestURL       *url.URL      `json:"requestURL"`       // Full URL of request
	RequestAddress   string        `json:"requestAddr"`      // Remote address from request
}

var NumberOfIterationsPerSec = 0

func init() {
	fmt.Println("Start INIT function")
	now := time.Now()

	duration := 1000 * int(time.Millisecond)
	iteration := 0

	fmt.Println(time.Now())

	for now.Add(time.Duration(duration)).After(time.Now()) {
		iteration += 1
	}

	NumberOfIterationsPerSec = iteration

	fmt.Println("# of runned iteration >", iteration, "/", NumberOfIterationsPerSec, "<")
	fmt.Println(time.Now())
}

func main() {
	var cfg = readConfigParameters()

	fmt.Printf("[MAIN]\t\tConfig values:\t%+v\n", cfg)

	var addr string = ":" + fmt.Sprint(cfg.Port)

	// start webserver
	for i, endpoint := range cfg.Endpoints {
		fmt.Printf("%d --> %s --> \n", i, endpoint)
		http.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
			// Set response writer to json
			w.Header().Set("Content-Type", "application/json")

			// Log request to stdout
			fmt.Printf("[REQUEST-INCOME] %s --> %s\n", r.URL, r.URL.Path)
			start := time.Now()
			//foundEndpoint := false

			for k, endp := range cfg.Endpoints {
				if endp == r.URL.Path {
					// Sleep not relevant here
					// time.Sleep(time.Duration(cfg.EndpointsDelay[k]) * time.Millisecond)

					// Get parameters to response
					response := Response{}
					response.ServiceName = cfg.Name
					response.ConfigOK = cfg.check()
					response.CalledEnpoint = endp
					response.CPU = int32(cfg.EndpointsCPU[k])
					response.Delay = int32(cfg.EndpointsDelay[k])
					response.CalloutParameter = cfg.EndpointsCall[k]

					// Remove ' character if get from command line
					call := strings.ReplaceAll(cfg.EndpointsCall[k], "'", "")
					// Check for empty/only whitespace string
					//length := len(strings.TrimSpace(call))

					// Split callout parameter by separate character: '__'
					calloutStringArray := strings.Split(call, "__")

					// Create array to collect callout responses
					calloutResponses := make([]string, len(calloutStringArray))

					// Make callout to async
					var waitgroupToCallouts sync.WaitGroup
					waitgroupToCallouts.Add(len(calloutStringArray))

					for i, callOut := range calloutStringArray {
						go calloutFunction(i, callOut, calloutResponses, &waitgroupToCallouts)
					}

					// for i, callOut := range calloutStringArray {
					// 	callOut = strings.ReplaceAll(callOut, "'", "")
					// 	fmt.Printf("[CALL_OUT]\t#no%d --> %s\n", i, callOut)
					// 	url := "http://" + callOut
					// 	resp, err := http.Get(url)

					// 	if err != nil {
					// 		calloutResponses = append(calloutResponses, "Oops, calling out failed")
					// 	} else {
					// 		// Convert response body to string
					// 		buf := new(strings.Builder)
					// 		_, err := io.Copy(buf, resp.Body)
					// 		if err != nil {
					// 			// Convertion failed
					// 			calloutResponses = append(calloutResponses, "Oops, failed to convert response to string")
					// 		} else {
					// 			// Convertion was successfull
					// 			calloutResponses = append(calloutResponses, string(buf.String()))
					// 		}

					// 	}

					// }

					// Generate CPU usage
					tightAlgorithm(int(cfg.EndpointsCPU[k]))

					// OLD: call CPU intense algorithm in different goroutines
					// Create waitgroup to wait all calculations done
					// var waitgroup sync.WaitGroup
					// waitgroup.Add(int(cfg.EndpointsCPU[k]))
					// for i := 0; i < int(cfg.EndpointsCPU[k]); i++ {
					// 	go AlgorithmToUseCPU(600, &waitgroup)
					// }
					// waitgroup.Wait()

					// Wait until all callout response get answer
					waitgroupToCallouts.Wait()
					// Then add to json
					response.Callouts = calloutResponses

					// After CPU calcualation wait if the delay time not passed
					waitTime := (time.Duration(cfg.EndpointsDelay[k]) * time.Millisecond) - time.Since(start)

					// Wait more if necessary
					if waitTime > 0 {
						time.Sleep(waitTime)
					}

					// Give more information about request/response
					response.Time = time.Now()
					response.RequestMethod = r.Method
					response.RequestURL = r.URL
					response.RequestAddress = r.RemoteAddr
					response.Host = r.Host
					response.ActualDelay = time.Since(start)

					json.NewEncoder(w).Encode(response)
				}

			}

			// send response time
			//fmt.Fprintf(w, "\nResponse time: %s\n", time.Now().Sub(start))
		})
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// log
		fmt.Printf("[REQUEST-INCOME] '%s' --> '%s'\n", r.URL, r.URL.Path)
		fmt.Printf("[MAIN]\t\tConfig values:\t%+v\n", cfg)
		fmt.Printf("%+v", r)
		// response
		fmt.Fprintf(w, "<h1>'/' or 404 page</h1>\n")
		fmt.Fprintf(w, "%+v", r)
	})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}

}

// Sieve of Eratosthenes
// func AlgorithmToUseCPU(number int, waitgroup *sync.WaitGroup) {
// 	max := number
// 	numbers := make([]bool, max+1)
// 	// Set values to ture
// 	for i := range numbers {
// 		numbers[i] = true
// 	}

// 	// main algorithm
// 	for p := 2; p*p <= max; p++ {
// 		if numbers[p] {
// 			for i := p * p; i <= max; i += p {

// 				numbers[i] = false
// 			}
// 		}
// 	}

// 	// Print prime numbers
// 	for p := 2; p <= max; p++ {
// 		if numbers[p] {
// 			fmt.Printf("%d ", p)
// 		}
// 	}

// 	waitgroup.Done()
// }

// Function to use CPU 100% to a specific duration
func tightAlgorithm(milliseconds int) {
	fmt.Println("Tight algoritm started")
	start := time.Now()

	duration := milliseconds * int(time.Millisecond)
	iterationNeeded := (NumberOfIterationsPerSec * milliseconds) / 1000
	
	iteration := 0
	dontCare := false


	for iteration < iterationNeeded {
		// time calculation need because init() function do the same in for loop
		dontCare = start.Add(time.Duration(duration)).After(time.Now())
		iteration += 1
	}

	fmt.Println("# of runned iteration (TightAlgorithm) >", iteration, "<", dontCare)

	fmt.Println("Running time: ", start, "-->", time.Now())

}

func calloutFunction(i int, callOut string, calloutResponses []string, waitgroupToCallouts *sync.WaitGroup) {
	fmt.Printf("\n[calloutFunction] New async callout started: '%d' --> '%s'\n", i, callOut)

	callOut = strings.ReplaceAll(callOut, "'", "")
	fmt.Printf("[CALL_OUT]\t#no%d --> %s\n", i, callOut)
	url := "http://" + callOut
	resp, err := http.Get(url)

	if err != nil {
		calloutResponses = append(calloutResponses, "Oops, calling out failed")
	} else {
		// Convert response body to string
		buf := new(strings.Builder)
		_, err := io.Copy(buf, resp.Body)
		if err != nil {
			// Convertion failed
			calloutResponses = append(calloutResponses, "Oops, failed to convert response to string")
		} else {
			// Convertion was successfull
			fmt.Printf("\n[calloutFunction] Async callout response: '%d' --> '%s'\n", i, string(buf.String()))
			calloutResponses = append(calloutResponses, string(buf.String()))
		}

	}

	waitgroupToCallouts.Done()
}