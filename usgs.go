package main

import (
	"fmt"
	"errors"
	"xml"
	"http"
	//    "json"
	//    "os"
	"strings"
)

type TimeSeriesResponse struct {
	TimeSeries []TimeSeries `xml:"timeSeries"`
}

type TimeSeries struct {
	Name string `xml:"attr"`
	// dunno why this didn't work
	//    SourceInfo SourceInfo `xml:"sourceInfo"`
	SiteName    string            `xml:"sourceInfo>siteName"`
	SiteCode    string            `xml:"sourceInfo>siteCode"`
	Code        string            `xml:"variable>variableCode"`
	Description string            `xml:"variable>variableDescription"`
	Values      []TimeSeriesValue `xml:"values>value"`
}

/* 
type SourceInfo struct {
    Name string `xml:"siteName"`
    Code string `xml:"siteCode"`
    Latitude float32 `xml:"geoLocation>geogLocation>latitude"`
    Longitude float32 `xml:"geoLocation>geogLocation>longitude"`
}
*/

type TimeSeriesValue struct {
	//    XMLName xml.Name `xml:"value"`
	DateTime string  `xml:"attr"`
	Value    float32 `xml:"chardata"`
}

type USGS_Source struct {
	url      string
	response TimeSeriesResponse
	config   []ConfigUSGS_Site
}

func NewUSGS_Source(config *Config) (usgs USGS_Source) {
	usgs.url = config.USGS_url
	usgs.config = config.Sources.USGS
	return
}

// return URL with add site=#,#,# & parameterCd=#,#,# paremeters
func (us *USGS_Source) buildUSGSQuery() (url string) {

	var sites []string
	site_map := map[string]bool{}
	var parameterCds []string
	cd_map := map[string]bool{}

	for _, site := range us.config {
		site_map[site.Site] = true
		cd_map[site.Param] = true
	}

	for site, _ := range site_map {
		sites = append(sites, site)
	}

	for cd, _ := range cd_map {
		parameterCds = append(parameterCds, cd)
	}

	return us.url + "&sites=" + strings.Join(sites, ",") + "&parameterCd=" + strings.Join(parameterCds, ",")

}

func (us *USGS_Source) FetchData(client *http.Client) (err error) {
	var resp *http.Response
	var timeSeriesResponse TimeSeriesResponse

	var prepared_url = us.buildUSGSQuery()
	resp, err = client.Get(prepared_url)
	if err != nil {
		return
	}
	err = xml.Unmarshal(resp.Body, &timeSeriesResponse)
	if err != nil {
		err = errors.New(fmt.Sprintf("Couldn't unmarshal XML: %s", err))
		return
	}

	us.response = timeSeriesResponse
	return

}

func (us *USGS_Source) Widgets() (widgets []string) {

	for _, site := range us.config {
		widgets = append(widgets, site.Widget)
		for _, bar := range [3]ConfigUSGS_Bar{site.Bars.Low, site.Bars.Current, site.Bars.High} {
			if bar.Widget != "" {
				widgets = append(widgets, bar.Widget)
			}
		}
	}
	return
}

func (us *USGS_Source) WidgetValue(widget_id string) (value int64, ts string, err error) {

	// return site config that matches widget id
	getSite := func() (site ConfigUSGS_Site, err error) {
		for _, s := range us.config {
			if widget_id == s.Widget {
				return s, err
			}
			for _, bar := range [3]ConfigUSGS_Bar{s.Bars.Low, s.Bars.Current, s.Bars.High} {
				if widget_id == bar.Widget {
					return s, err
				}
			}
		}
		err = errors.New("no match in config")
		return
	}

	var site ConfigUSGS_Site
	site, err = getSite()
	if err != nil {
		return
	}

	// if this is a static value defined in the config
	for _, bar := range [3]ConfigUSGS_Bar{site.Bars.Low, site.Bars.High} {
		if widget_id == bar.Widget {
			value = int64(bar.Value)
			return
		}
	}

	// is value for widget is in response
	for _, series := range us.response.TimeSeries {
		if site.Site == series.SiteCode && site.Param == series.Code {
			value = int64(series.Values[0].Value)
			ts = series.Values[0].DateTime
			return
		}
	}
	err = errors.New("no match in response")
	return
}
