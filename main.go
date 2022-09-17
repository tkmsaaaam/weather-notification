package main

import (
	"fmt"
	"github.com/slack-go/slack"
	"github.com/tkmsaaaam/weather-api-go"
	"os"
	"strings"
)

func getWeather() string {
	body, err := weather.Get(os.Args[1])
	if err != nil {
		fmt.Println("Error Request API")
	}

	var todayForecast weather.Forecast

	for _, forecast := range body.Forecasts {
		dateLabel := forecast.DateLabel
		if dateLabel == "今日" {
			todayForecast = forecast
		}
	}

	lastN := strings.LastIndex(body.Description.Text, "\n") + 1

	mark := "\n----------\n"
	message := mark + "日時:" + body.PublicTimeFormatted + "\n概要:" + body.Description.Text[0:lastN] + "\n最低気温:" + todayForecast.Temperature.Min.Celsius + "\n最高気温:" + todayForecast.Temperature.Max.Celsius + "\n" + "\n00-06:" + todayForecast.ChanceOfRain.T0006 + "\n06-12:" + todayForecast.ChanceOfRain.T0612 + "\n12-18:" + todayForecast.ChanceOfRain.T1218 + "\n18-24:" + todayForecast.ChanceOfRain.T1824 + mark
	return message
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
	message := getWeather()
	postSlack(message)
}
