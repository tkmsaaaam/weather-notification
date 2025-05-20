package main

import (
	"log"
	"net/url"
	"os"
	"regexp"
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

type ChanceOfRain struct {
	T0006 float64
	T0612 float64
	T1218 float64
	T1824 float64
}

type Result struct {
	min          float64
	max          float64
	ChanceOfRain *ChanceOfRain
	wind         float64
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
	max, _ := strconv.ParseFloat(todayForecast.Temperature.Max.Celsius, 64)
	min, _ := strconv.ParseFloat(todayForecast.Temperature.Min.Celsius, 64)
	t0006, _ := strconv.ParseFloat(strings.ReplaceAll(todayForecast.ChanceOfRain.T0006, "%", ""), 64)
	t0612, _ := strconv.ParseFloat(strings.ReplaceAll(todayForecast.ChanceOfRain.T0612, "%", ""), 64)
	t1218, _ := strconv.ParseFloat(strings.ReplaceAll(todayForecast.ChanceOfRain.T1218, "%", ""), 64)
	t1824, _ := strconv.ParseFloat(strings.ReplaceAll(todayForecast.ChanceOfRain.T1824, "%", ""), 64)
	chanceOfRain := ChanceOfRain{
		T0006: t0006,
		T0612: t0612,
		T1218: t1218,
		T1824: t1824,
	}
	windStr := regexp.MustCompile("[0-9]+").FindString(todayForecast.Detail.Wind)
	wind, _ := strconv.ParseFloat(windStr, 64)
	return &message, &Result{max: max, min: min, ChanceOfRain: &chanceOfRain, wind: wind}
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
	log.Println(*message)
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
	pusher.send("temperature_max", "maximum temperature", result.max, map[string]string{"type": "temperature"})
	pusher.send("temperature_min", "minimum temperature", result.min, map[string]string{"type": "temperature"})
	pusher.send("chance_of_rain_t0006", "chance of rain", result.ChanceOfRain.T0006/100, map[string]string{"type": "rain"})
	pusher.send("chance_of_rain_t0612", "chance of rain", result.ChanceOfRain.T0612/100, map[string]string{"type": "rain"})
	pusher.send("chance_of_rain_t1218", "chance of rain", result.ChanceOfRain.T1218/100, map[string]string{"type": "rain"})
	pusher.send("chance_of_rain_t1824", "chance of rain", result.ChanceOfRain.T1824/100, map[string]string{"type": "rain"})
	pusher.send("wind", "wind", result.wind, map[string]string{"type": "wind"})
}

type Pusher struct {
	*push.Pusher
}

func (pusher Pusher) send(k, description string, v float64, labels map[string]string) {
	labels["pusher"] = "weather"
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "weather",
		Name:        k,
		Help:        description,
		ConstLabels: labels,
	})
	gauge.Set(v)
	if err := pusher.Collector(gauge).Push(); err != nil {
		log.Println("can not push", k, err)
	}
}
