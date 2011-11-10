package ducksboard

import (
	"bytes"
	"http"
    "json"
)

const PUSH_URL = "https://push.ducksboard.com/values/"

type Counter struct {
    Timestamp int64 `json:"timestamp"`
    Value int `json:"value"`
}

type Gauge struct {
    Timestamp int64 `json:"timestamp"`
    Value float32 `json:"value"`
}

type Graph struct {
    Timestamp int64 `json:"timestamp"`
    Value int `json:"value"`
}

type Bar struct {
    Timestamp int64 `json:"timestamp"`
    Value int `json:"value"`
}

type Box struct {
    Timestamp int64 `json:"timestamp"`
    Value int `json:"value"`
}

type Pin struct {
    Timestamp int64 `json:"timestamp"`
    Value int `json:"value"`
}

type Image struct {
    Timestamp int64 `json:"timestamp"`
    Value ImageValue `json:"value"`
}

type ImageValue struct {
    Source string `json:"source"`
    Caption string `json:"caption"`
}

type Timeline struct {
    Timestamp int64 `json:"timestamp"`
    Value TimelineValue `json:"value"`
}

type TimelineValue struct {
    Title string `json:"title"`
    Image string `json:"image"`
    Content string `json:"content"`
    Source string `json:"source"`
    Link string `json:"link"`
}

type PushRequest struct {
    WidgetID string
    APIkey string
    Value interface{}
}

func NewPushRequest(apikey string) (*PushRequest) {
   req := new(PushRequest)
   req.APIkey = apikey
   return req
}

func (pr *PushRequest) Request() (req *http.Request, err error) {
    var b []byte

    b,err = json.Marshal(pr.Value)
    if err != nil {
        return
    }
    buf := bytes.NewBuffer(b)

    req, err = http.NewRequest("POST", PUSH_URL + pr.WidgetID, buf)
    if err != nil {
        return
    }
    req.SetBasicAuth(pr.APIkey,"ignored")
    req.Header.Add("Content-type","application/x-www-form-urlencoded")
    return
}
