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
    Refresh int `json:"refresh"`
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
        fmt.Printf("Invalid config: %s\n", err)
        return
    }

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

    // Image Updates
    for _,image := range(config.Sources.Image) {
        push_rq := ducksboard.NewPushRequest(image.Widget,config.API_key)
        db := ducksboard.Image{Timestamp: now}
        db.Value.Source = image.Source
        db.Value.Caption = image.Caption
        json,err := json.Marshal(db)
        if err != nil {
            fmt.Printf("error marshalling db_val: %s", err)
            break;
        }

        push_rq.Value = string(json)
        http_req,_ := push_rq.Request()
        clientRequests <- http_req
    }


    // USGS Updates
    usgs := NewUSGS_Source(config)

    err = usgs.FetchData(http_client)
    if err != nil {
        fmt.Printf("error fetching data: %s", err)
        return
    }
    //fmt.Printf("USGS Response: %s\n", usgs.response)

    for _,widget_id := range(usgs.Widgets()) {
        val,ts,err := usgs.WidgetValue(widget_id)
        if err != nil {
            fmt.Printf("error getting value for widget, %s\n", err)
        } else {
            push_rq := ducksboard.NewPushRequest(widget_id,config.API_key)
            db := ducksboard.Counter{Value: int(val) }

            t,err := time.Parse(time.RFC3339,ts)
            if err == nil {
                db.Timestamp = t.Seconds()
            }

            json,err := json.Marshal(db)
            if err != nil {
                fmt.Printf("error marshalling db_val: %s", err)
                break;
            }

            push_rq.Value = string(json)
            http_req,_ := push_rq.Request()
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
