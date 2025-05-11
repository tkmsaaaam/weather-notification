package main

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/slack-go/slack"
	"github.com/tkmsaaaam/weather-api-go"
)

type SlackClient struct {
	*slack.Client
}

type WeatherClient struct {
	weather.Client
}

type Result struct {
	min int
	max int
}

func (weatherClient WeatherClient) getWeather() (*string, *Result) {
	body, err := weatherClient.Get(os.Getenv("CITY_ID"))
	if err != nil {
		log.Printf("Error Request API %v\n", err)
		return nil, nil
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
	max, _ := strconv.Atoi(todayForecast.Temperature.Max.Celsius)
	min, _ := strconv.Atoi(todayForecast.Temperature.Min.Celsius)
	return &message, &Result{max: max, min: min}
}

func (client SlackClient) postSlack(message string) {
	_, _, err := client.PostMessage(os.Getenv("SLACK_CHANNEL_ID"), slack.MsgOptionText(message, true))
	if err != nil {
		log.Printf("can not post. %v\n", err)
	}
}

func main() {
	weatherClient := weather.New()
	message, result := WeatherClient{weatherClient}.getWeather()
	client := slack.New(os.Getenv("SLACK_BOT_TOKEN"))
	SlackClient{client}.postSlack(*message)
	otelExporterEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_METRICS_ENDPOINT")
	if otelExporterEndpoint == "" {
		// OTEL_EXPORTER_OTLP_METRICS_ENDPOINT is optional, so no need to log
		return
	}
	_, err := url.Parse(otelExporterEndpoint)
	if err != nil {
		log.Println("can not parse otel url:", err)
		return
	}
	pusher := Pusher{push.New(otelExporterEndpoint, "weather")}
	pusher.send("max", "temperature", result.max)
	pusher.send("min", "temperature", result.min)
}

type Pusher struct {
	*push.Pusher
}

func (pusher Pusher) send(k, grouping string, v int) {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   "weather",
		Name:        k,
		Help:        k + " by " + grouping,
		ConstLabels: prometheus.Labels{"pusher": "weather", "grouping": grouping},
	})
	counter.Add(float64(v))
	if err := pusher.Collector(counter).Push(); err != nil {
		log.Println("can not push", k, err)
	}
}
