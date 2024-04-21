package main

import (
	"log"
	"os"
	"strings"

	"github.com/slack-go/slack"
	"github.com/tkmsaaaam/weather-api-go"
)

type SlackClient struct {
	*slack.Client
}

type WeatherClient struct {
	weather.Client
}

func (weatherClient WeatherClient) getWeather() *string {
	body, err := weatherClient.Get(os.Getenv("CITY_ID"))
	if err != nil {
		log.Printf("Error Request API %v\n", err)
		return nil
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
	return &message
}

func (client SlackClient) postSlack(message string) {
	_, _, err := client.PostMessage(os.Getenv("SLACK_CHANNEL_ID"), slack.MsgOptionText(message, true))
	if err != nil {
		log.Printf("can not post. %v\n", err)
	}
}

func main() {
	weatherClient := weather.New()
	message := WeatherClient{weatherClient}.getWeather()
	client := slack.New(os.Getenv("SLACK_BOT_TOKEN"))
	SlackClient{client}.postSlack(*message)
}
