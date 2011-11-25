package main

import (
	"encoding/json"
	"io/ioutil"
)

const MAX_PUSH_CONNECTIONS = 4

type Config struct {
	Push_url string        `json:"push_url"`
	API_key  string        `json:"api_key"`
	Refresh  int64         `json:"refresh"`
	USGS_url string        `json:"usgs_url"`
	Sources  ConfigSources `json:"sources"`
}

type ConfigSources struct {
	Image []ConfigImage     `json:"image"`
	USGS  []ConfigUSGS_Site `json:usgs`
}

type ConfigImage struct {
	Source  string `json:"source"`
	Caption string `json:"caption"`
	Widget  string `json:"widget"`
}

type ConfigUSGS_Site struct {
	Site   string          `json:"site"`
	Param  string          `json:"param"`
	Widget string          `json:"widget"`
	Bars   ConfigUSGS_Bars `json:"bars,omitempty"`
}

type ConfigUSGS_Bars struct {
	Low     ConfigUSGS_Bar `json:"low"`
	Current ConfigUSGS_Bar `json:"current"`
	High    ConfigUSGS_Bar `json:"high"`
}

type ConfigUSGS_Bar struct {
	Widget string `json:"widget"`
	Value  int    `json:"value"`
}

func ParseConfig(filename string) (config *Config, err error) {
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
