package main

// I'm a long standing C developer learning Go
// Any comments on more idiomatic Go are welcome

// Prometheus Exporter for Elgato (Keylight at first, more to come)

// Example output
// {"numberOfLights":1,"lights":[{"on":1,"brightness":55,"temperature":198}]}

// TODO - Need the ability to set labels for a particular light (IP/port) to give it a location

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// JSON structures:
//
// {"numberOfLights":1,"lights":[{"on":1,"brightness":55,"temperature":198}]}

type elgatoLight struct {
	On       int
	Brightness  int
	Temperature int
}

type elgatoResponse struct {
	NumberOfLights int
	Lights         []elgatoLight
}

var (
	hc       *http.Client
	cfg      Config
	recorder Recorder
)

type Config struct {
	timeout      time.Duration // timeout in msec
	ipaddress    string        // IP address
	port         int           // port
	metricport   int           // metricport
	pollinterval time.Duration // polling interval
	pollurl      string        // url for polling
	metricurl    string        // url for metrics
	datastore    string        // datastore ("" means no storage)
	file         string        // filename to parse straight away
}

func RegisterFlags() {
	flag.DurationVar(&cfg.timeout, "timeout", 1*time.Second, "Timeout for polling light")
	flag.StringVar(&cfg.ipaddress, "ipaddress", "192.168.1.209", "IP Address of light")
	flag.IntVar(&cfg.port, "port", 9123, "Port of light")
	flag.IntVar(&cfg.metricport, "metricport", 9091, "port for serving metrics")
	flag.DurationVar(&cfg.pollinterval, "interval", 10*time.Second, "Polling interval")
	flag.StringVar(&cfg.pollurl, "pollurl", "elgato/lights", "URL to poll")
	flag.StringVar(&cfg.metricurl, "metricurl", "/metrics", "URL to server metrics")
	flag.StringVar(&cfg.datastore, "datastore", "", "Datastore directory, blank to disable")
	flag.StringVar(&cfg.file, "file", "", "Parse file directly")
}

func parseJSON(body string) error {
	var r elgatoResponse
	parseTime := time.Now()
	err := json.Unmarshal([]byte(body), &r)
	durParse := time.Since(parseTime)

	recorder.measureParseDur(durParse)

	if err != nil {
		// TODO - do something with err.Error() ?
		recorder.measureLastError(parseTime)
		log.Fatal("parsing:", err)
		return err
	}

	// TODO - check values are within bounds?

	// TODO - deal with numbers of lights?

	recorder.measureOnOff(r.Lights[0].On)
	recorder.measureBrightness(r.Lights[0].Brightness)
	recorder.measureTemperature(r.Lights[0].Temperature)

	return nil
}

func doPoll() {
	url := fmt.Sprintf("http://%s:%d/%s", cfg.ipaddress, cfg.port, cfg.pollurl)
	pollTime := time.Now()
	req, err := http.NewRequest("GET", url, nil)
	resp, err := hc.Do(req)
	durPoll := time.Since(pollTime)

	recorder.measureLastPoll(pollTime)
	recorder.measurePollDur(durPoll)

	// fmt.Printf("statusCode=%d\n", resp.StatusCode)

	if err != nil {
		fmt.Printf("Got error [%s]\n", err)
		// TODO - do anything with err.Error() ?
		recorder.measureLastError(pollTime)
		recorder.measurePolls("error")
		return
	}
	// Got a good response!
	defer resp.Body.Close()

	// Log the statusCode
	recorder.measureStatusCode(resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// TODO
		recorder.measurePolls("error-readall")
		return
	}

	recorder.measureLastGoodPoll(pollTime)

	// Stash the downloaded file if we have a datastore configured
	if cfg.datastore != "" {
		fname := cfg.datastore + "/" + pollTime.Format("20060102150405.000000")
		os.WriteFile(fname, body, 0644)
	}
	// parse file and update values
	parseTime := time.Now()
	err = parseJSON(string(body))
	durParse := time.Since(parseTime)
	recorder.measureParseDur(durParse)

	if err != nil {
		// TODO Do something
		recorder.measureLastError(pollTime)
		recorder.measurePolls("parse-error")
		return
	}

	recorder.measurePolls("ok")
}

func main() {
	RegisterFlags()
	flag.Parse()

	recorder = NewRecorder(prometheus.DefaultRegisterer) // TODO - prefix

	// check if we are parsing a single file
	if cfg.file != "" {
		// TODO read in file
		body, err := os.ReadFile(cfg.file)
		if err != nil {
			log.Fatal("ReadFile:", err)
		}
		// parse it
		err = parseJSON(string(body))
		if err != nil {
			log.Fatal("parseJSON:", err)
		}
		// output data
		// output := metricText()
		// fmt.Printf("%s", output)
		return
	}

	// Serve Prom metrics on cfg.metricport
	go func() {
		listenaddr := ":" + strconv.Itoa(cfg.metricport)
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(listenaddr, nil)
		if err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	// Setup timeout on http client
	hc = &http.Client{
		Timeout: cfg.timeout,
	}

	// Infinite loop of polling
	for {
		doPoll()
		time.Sleep(cfg.pollinterval)
	}
}
