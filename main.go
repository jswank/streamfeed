package main

import (
	"fmt"
	"http"
    "json"
    "time"
    "io/ioutil"
    "flag"
    "log"
    "./ducksboard/_obj/ducksboard"
)

const MAX_PUSH_CONNECTIONS = 4

type Config struct {
    Push_url string `json:"push_url"`
    API_key string `json:"api_key"`
    Refresh int64 `json:"refresh"`
    USGS_url string `json:"usgs_url"`
    Sources ConfigSources `json:"sources"`
}

type ConfigSources struct {
    Image []ConfigImage `json:"image"`
    USGS []ConfigUSGS_Site `json:usgs`
}

type ConfigImage struct {
    Source string `json:"source"`
    Caption string `json:"caption"`
    Widget string `json:"widget"`
}

type ConfigUSGS_Site struct {
    Site string `json:"site"`
    Param string `json:"param"`
    Widget string `json:"widget"`
    Bars ConfigUSGS_Bars `json:"bars,omitempty"`
}

type ConfigUSGS_Bars struct {
    Low ConfigUSGS_Bar `json:"low"`
    Current ConfigUSGS_Bar `json:"current"`
    High ConfigUSGS_Bar `json:"high"`
}

type ConfigUSGS_Bar struct {
    Widget string `json:"widget"`
    Value int `json:"value"`
}

func ParseConfig(filename string) (config Config, err error) {
    var b []byte

    b, err = ioutil.ReadFile(filename)
    if err != nil {
        return
    }
    err = json.Unmarshal(b, &config)
    if err != nil {
        return
    }
    return
}

func LoopUpdate(config Config, quit chan bool) {
    Update(config)
    if ( config.Refresh < 60 ) {
        log.Println("Refresh interval is less than 60 seconds. Exiting to avoid abuse...")
        quit<-true
    }
    for {
        log.Printf("Sleeping for %d seconds\n", config.Refresh)
        time.Sleep(config.Refresh*1e9)
        Update(config)
    }
}

func Update(config Config) {

    http_client := new(http.Client)

    // this is the channel that receives http requests to ducksboard
    var clientRequests = make(chan *http.Request)

    // Push routines talk back on this channel
    var push_complete = make(chan int)
    for i:= 0; i < MAX_PUSH_CONNECTIONS; i++ {
        go Push(http_client,push_complete,clientRequests)
    }

    // current time
    now := time.UTC().Seconds()

    push_rq := ducksboard.NewPushRequest(config.API_key)

    // Image Updates
    log.Printf("Pushing image widgets\n")
    for _,image := range(config.Sources.Image) {
        rq_val := ducksboard.Image{Timestamp: now}
        rq_val.Value.Source = image.Source
        rq_val.Value.Caption = image.Caption

        push_rq.Value = rq_val
        push_rq.WidgetID = image.Widget

        http_req,err := push_rq.Request()
        if err != nil {
            log.Printf("error generating push request, %s", err)
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
    //fmt.Printf("USGS Response: %s\n", usgs.response)

    log.Printf("Pushing USGS widgets\n")
    for _,widget_id := range(usgs.Widgets()) {
        val,ts,err := usgs.WidgetValue(widget_id)
        if err != nil {
            log.Printf("error getting value for widget, %s\n", err)
        } else {
 //           push_rq := ducksboard.NewPushRequest(widget_id,config.API_key)
            rq_val := ducksboard.Counter{Value: int(val)}

            t,err := time.Parse(time.RFC3339,ts)
            if err == nil {
                rq_val.Timestamp = t.Seconds()
            }

            push_rq.Value = rq_val
            push_rq.WidgetID = widget_id

            http_req,err := push_rq.Request()
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
    for i:= 0; i < MAX_PUSH_CONNECTIONS; i++ {
        <-push_complete
    }

}

func Push(client *http.Client, quit chan int, queue chan *http.Request) {
    for r:= range queue {
        resp,err := client.Do(r)
        if err != nil {
            log.Printf("error performing push: %s", err)
        } else {
            resp.Body.Close() // to allow re-use of underlying TCP connection w/ http keep-alives
            //log.Printf("response: %s", resp)
        }
    }
    quit<-1
    return
}

func main() {

    var config_file = flag.String("f", "config.json", "configuration file")
    flag.Parse()

    config,err := ParseConfig(*config_file)
    if err != nil {
        log.Fatal( fmt.Sprintf("Invalid config: %s\n", err) )
    }

    // this is the channel that main blocks on
    var quit = make(chan bool)

    go LoopUpdate(config,quit)

    <-quit

}
