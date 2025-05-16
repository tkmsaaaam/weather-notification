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

type ChanceOfRain struct {
	T0006 int
	T0612 int
	T1218 int
	T1824 int
}

type Result struct {
	min          int
	max          int
	ChanceOfRain *ChanceOfRain
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
	t0006, _ := strconv.Atoi(strings.ReplaceAll(todayForecast.ChanceOfRain.T0006, "%", ""))
	t0612, _ := strconv.Atoi(strings.ReplaceAll(todayForecast.ChanceOfRain.T0612, "%", ""))
	t1218, _ := strconv.Atoi(strings.ReplaceAll(todayForecast.ChanceOfRain.T1218, "%", ""))
	t1824, _ := strconv.Atoi(strings.ReplaceAll(todayForecast.ChanceOfRain.T1824, "%", ""))
	chanceOfRain := ChanceOfRain{
		T0006: t0006,
		T0612: t0612,
		T1218: t1218,
		T1824: t1824,
	}
	return &message, &Result{max: max, min: min, ChanceOfRain: &chanceOfRain}
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
	pusher.send("temperature_max", "maximum temperature", result.max, map[string]string{"type": "temperature"})
	pusher.send("temperature_min", "minimum temperature", result.min, map[string]string{"type": "temperature"})
	pusher.send("chance_of_rain_t0006", "chance of rain", result.ChanceOfRain.T0006, map[string]string{"type": "rain"})
	pusher.send("chance_of_rain_t0612", "chance of rain", result.ChanceOfRain.T0612, map[string]string{"type": "rain"})
	pusher.send("chance_of_rain_t1218", "chance of rain", result.ChanceOfRain.T1218, map[string]string{"type": "rain"})
	pusher.send("chance_of_rain_t1824", "chance of rain", result.ChanceOfRain.T1824, map[string]string{"type": "rain"})
}

type Pusher struct {
	*push.Pusher
}

func (pusher Pusher) send(k, description string, v int, labels map[string]string) {
	labels["pusher"] = "weather"
	label := labels
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "weather",
		Name:        k,
		Help:        description,
		ConstLabels: label,
	})
	gauge.Add(float64(v))
	if err := pusher.Collector(gauge).Push(); err != nil {
		log.Println("can not push", k, err)
	}
}
