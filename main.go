package main

import (
	"encoding/json"
	"fmt"
	"github.com/slack-go/slack"
	"github.com/tkmsaaaam/weather-api-go"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func requestApi(url string) ([]byte, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	client := new(http.Client)
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error Request:", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		fmt.Println("Error Response:", resp.Status)
		return nil, err
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatalf("ioutil.ReadAll err=%s", readErr.Error())
		return nil, readErr
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

	var res weather.Response

	if err = json.Unmarshal(body, &res); err != nil {
		log.Fatalf("json.Unmarshal err=%s", err.Error())
	}

	forecasts := res.Forecasts

	var todayForecast weather.Forecast

	for _, forecast := range forecasts {
		dateLabel := forecast.DateLabel
		if dateLabel == "今日" {
			todayForecast = forecast
		}
	}

	lastN := strings.LastIndex(res.Description.Text, "\n") + 1

	mark := "\n----------\n"
	message := mark + "日時:" + res.PublicTimeFormatted + "\n概要:" + res.Description.Text[0:lastN] + "\n最低気温:" + todayForecast.Temperature.Min.Celsius + "\n最高気温:" + todayForecast.Temperature.Max.Celsius + "\n" + "\n00-06:" + todayForecast.ChanceOfRain.T0006 + "\n06-12:" + todayForecast.ChanceOfRain.T0612 + "\n12-18:" + todayForecast.ChanceOfRain.T1218 + "\n18-24:" + todayForecast.ChanceOfRain.T1824 + mark

	postSlack(message)
}
