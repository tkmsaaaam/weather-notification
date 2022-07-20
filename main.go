package main

import (
	"encoding/json"
	"fmt"
	"github.com/slack-go/slack"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type Description struct {
	PublicTime          time.Time `json:"publicTime"`
	PublicTimeFormatted string    `json:"publicTimeFormatted"`
	HeadlineText        string    `json:"headlineText"`
	BodyText            string    `json:"bodyText"`
	Text                string    `json:"text"`
}

type Detail struct {
	Weather string `json:"weather"`
	Wind    string `json:"wind"`
	Wave    string `json:"wave"`
}

type DetailTemperature struct {
	Celsius    string `json:"celsius"`
	Fahrenheit string `json:"fahrenheit"`
}

type Temperature struct {
	Min DetailTemperature `json:"min"`
	Max DetailTemperature `json:"max"`
}

type ChanceOfRain struct {
	T0006 string `json:"T00_06"`
	T0612 string `json:"T06_12"`
	T1218 string `json:"T12_18"`
	T1824 string `json:"T18_24"`
}

type Image struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Forecast struct {
	Date         string       `json:"date"`
	DateLabel    string       `json:"dateLabel"`
	Telop        string       `json:"telop"`
	Detail       Detail       `json:"detail"`
	Temperature  Temperature  `json:"temperature"`
	ChanceOfRain ChanceOfRain `json:"chanceOfRain"`
	Image        Image        `json:"image"`
}

type Location struct {
	Area       string `json:"area"`
	Prefecture string `json:"prefecture"`
	District   string `json:"district"`
	City       string `json:"city"`
}

type CopyrightImage struct {
	Title  string `json:"title"`
	Link   string `json:"link"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Provider struct {
	Link string `json:"link"`
	Name string `json:"name"`
	Note string `json:"note"`
}

type Copyright struct {
	Title    string         `json:"title"`
	Link     string         `json:"link"`
	Image    CopyrightImage `json:"image"`
	Provider []Provider     `json:"provider"`
}

type Response struct {
	PublicTime          time.Time   `json:"publicTime"`
	PublicTimeFormatted string      `json:"publicTimeFormatted"`
	PublishingOffice    string      `json:"publishingOffice"`
	Title               string      `json:"title"`
	Link                string      `json:"link"`
	Description         Description `json:"description"`
	Forecasts           []Forecast  `json:"forecasts"`
	Location            Location    `json:"location"`
	Copyright           Copyright   `json:"copyright"`
}

func requestApi(url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	client := new(http.Client)
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error Request:", err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Error Response:", resp.Status)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("ioutil.ReadAll err=%s", err.Error())
	}

	return body, nil
}

func postSlack(message string) {
	token := os.Args[2]
	c := slack.New(token)

	_, _, err := c.PostMessage(os.Args[3], slack.MsgOptionText(message, true))
	if err != nil {
		panic(err)
	}
}

func main() {
	baseUrl := "https://weather.tsukumijima.net/api/forecast/city/" + os.Args[1]
	body, err := requestApi(baseUrl)

	if err != nil {
		fmt.Println("Error Request API")
	}

	var res Response

	if err = json.Unmarshal(body, &res); err != nil {
		log.Fatalf("json.Unmarshal err=%s", err.Error())
	}

	forecasts := res.Forecasts

	var todayForecast Forecast

	for _, forecast := range forecasts {
		dateLabel := forecast.DateLabel
		if dateLabel == "今日" {
			todayForecast = forecast
		}
	}

	message := "日時:" + res.PublicTimeFormatted + "\n概要:" + res.Description.Text + "\n最低気温:" + todayForecast.Temperature.Min.Celsius + "\n最高気温:" + todayForecast.Temperature.Max.Celsius + "\n" + "\n00-06:" + todayForecast.ChanceOfRain.T0006 + "\n06-12:" + todayForecast.ChanceOfRain.T0612 + "\n12-18:" + todayForecast.ChanceOfRain.T1218 + "\n18-24:" + todayForecast.ChanceOfRain.T1824

	postSlack(message)
}
