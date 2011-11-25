package main

import (
	"./ducksboard/_obj/ducksboard"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Run Update() at a regular interval
func LoopUpdate(config *Config, quit chan bool) {
	Update(config)
	if config.Refresh < 60 {
		log.Println("Refresh interval is less than 60 seconds. Exiting to avoid abuse...")
		quit <- true
	}
	for {
		log.Printf("Sleeping for %d seconds\n", config.Refresh)
		time.Sleep(config.Refresh * 1e9)
		Update(config)
	}
}

func buildImageReq(push_rq *ducksboard.PushRequest, image *ConfigImage) (http_req *http.Request, err error) {

	rq_val := ducksboard.Image{Timestamp: time.UTC().Seconds()}
	rq_val.Value.Source = image.Source
	rq_val.Value.Caption = image.Caption

	push_rq.Value = rq_val
	push_rq.WidgetID = image.Widget

	http_req, err = push_rq.Request()
	return
}

func Update(config *Config) {

	// a single HTTP client is used
	http_client := new(http.Client)

	// this is the channel that receives http requests to ducksboard
	var clientRequests = make(chan *http.Request)

	// Push routines talk back on this channel
	var push_complete = make(chan int)
	for i := 0; i < MAX_PUSH_CONNECTIONS; i++ {
		go Push(http_client, push_complete, clientRequests)
	}

	// a single PushRequest is re-used
	push_rq := ducksboard.NewPushRequest(config.API_key)

	// Image Updates
	log.Printf("Pushing image widgets\n")
	for _, image := range config.Sources.Image {
		http_req, err := buildImageReq(push_rq, &image)
		if err != nil {
			log.Printf("error building image push request, %s", err)
			continue
		}
		clientRequests <- http_req
	}

	// USGS Updates
	usgs := NewUSGS_Source(config)
	log.Printf("Fetching USGS data\n")
	err := usgs.FetchData(http_client)
	if err != nil {
		log.Printf("error fetching data: %s", err)
		return
	}

	log.Printf("Pushing USGS widgets\n")
	for _, widget_id := range usgs.Widgets() {
		val, ts, err := usgs.WidgetValue(widget_id)
		if err != nil {
			log.Printf("error getting value for widget, %s\n", err)
		} else {
			//           push_rq := ducksboard.NewPushRequest(widget_id,config.API_key)
			rq_val := ducksboard.Counter{Value: int(val)}

			t, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				rq_val.Timestamp = t.Seconds()
			}

			push_rq.Value = rq_val
			push_rq.WidgetID = widget_id

			http_req, err := push_rq.Request()
			if err != nil {
				log.Printf("error generating push request, %s", err)
				continue
			}
			clientRequests <- http_req
		}
	}

	// all requests have been placed in the channel
	close(clientRequests)

	// block until all Push routines to complete
	for i := 0; i < MAX_PUSH_CONNECTIONS; i++ {
		<-push_complete
	}

}

// execute HTTP request
func Push(client *http.Client, quit chan int, queue chan *http.Request) {
	for r := range queue {
		resp, err := client.Do(r)
		// Body is always non-nil: this works regardless of error
		resp.Body.Close()

		if err != nil {
			log.Printf("error performing push: %s", err)
		} else if resp.StatusCode != 200 {
			log.Printf("got non-OK response from Ducksboard API: %s", resp.Status)
		}
	}
	quit <- 1
	return
}

func main() {

	var config_file = flag.String("f", "config.json", "configuration file")
	flag.Parse()

	config, err := ParseConfig(*config_file)
	if err != nil {
		log.Fatal(fmt.Sprintf("Invalid config: %s\n", err))
	}

	// this is the channel that main() blocks on
	var quit = make(chan bool)

	go LoopUpdate(config, quit)

	<-quit

}
