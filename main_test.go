package main

import (
	"bytes"
	"embed"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slacktest"
	"github.com/tkmsaaaam/weather-api-go"
)

//go:embed testdata/*
var testdata embed.FS

func TestGetWeather(t *testing.T) {
	type res struct {
		status int
		body   string
	}
	weatherTokyo, _ := testdata.ReadFile("testdata/weather/TOKYO.json")

	tests := []struct {
		name   string
		apiRes res
		want   *weather.Forecast
	}{
		{
			name:   "watherIsOk",
			apiRes: res{status: 200, body: string(weatherTokyo)},
			want: &weather.Forecast{
				Date:      "2023-05-19",
				DateLabel: "今日",
				Temperature: weather.Temperature{
					Min: weather.DetailTemperature{
						Celsius: "0",
					},
					Max: weather.DetailTemperature{
						Celsius: "30",
					},
				},
				ChanceOfRain: weather.ChanceOfRain{
					T0006: "--%",
					T0612: "00%",
					T1218: "50%",
					T1824: "70%",
				},
			},
		},
		{
			name:   "apiIsError",
			apiRes: res{status: 500, body: "Internal Server Error"},
			want:   nil,
		},
	}
	for _, tt := range tests {
		TOKYO := "130010"
		t.Setenv("CITY_ID", TOKYO)
		mux := http.NewServeMux()
		mux.HandleFunc("/api/forecast/city/"+TOKYO, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tt.apiRes.status)
			io.WriteString(w, tt.apiRes.body)
		})
		weatherClient := WeatherClient{weather.Client{Client: &http.Client{Transport: localRoundTripper{handler: mux}}}}

		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defaultFlags := log.Flags()
			log.SetFlags(0)
			defer func() {
				log.SetOutput(os.Stderr)
				log.SetFlags(defaultFlags)
				buf.Reset()
			}()
			_, got := weatherClient.getWeather()
			if tt.want == nil && got != nil || tt.want != nil && got == nil {
				t.Errorf("getWeather() = %v, want %v", got, tt.want)
			}
			if got != nil || tt.want != nil {
				if got.Date != tt.want.Date {
					t.Errorf("getWeather().Date = %v, want %v", got.Date, tt.want.Date)
				}
				if got.DateLabel != tt.want.DateLabel {
					t.Errorf("getWeather().DateLabel = %v, want %v", got.DateLabel, tt.want.DateLabel)
				}
				if got.Temperature.Min.Celsius != tt.want.Temperature.Min.Celsius {
					t.Errorf("getWeather().Temperature.Min.Celsius = %v, want %v", got.DateLabel, tt.want.DateLabel)
				}
				if got.Temperature.Max.Celsius != tt.want.Temperature.Max.Celsius {
					t.Errorf("getWeather().Temperature.Min.Celsius = %v, want %v", got.DateLabel, tt.want.DateLabel)
				}
				if got.ChanceOfRain.T0006 != tt.want.ChanceOfRain.T0006 {
					t.Errorf("getWeather().ChanceOfRain.T0006 = %v, want %v", got.ChanceOfRain.T0006, tt.want.ChanceOfRain.T0006)
				}
				if got.ChanceOfRain.T0612 != tt.want.ChanceOfRain.T0612 {
					t.Errorf("getWeather().ChanceOfRain.T0612 = %v, want %v", got.ChanceOfRain.T0612, tt.want.ChanceOfRain.T0612)
				}
				if got.ChanceOfRain.T1218 != tt.want.ChanceOfRain.T1218 {
					t.Errorf("getWeather().ChanceOfRain.T1218 = %v, want %v", got.ChanceOfRain.T1218, tt.want.ChanceOfRain.T1218)
				}
				if got.ChanceOfRain.T1824 != tt.want.ChanceOfRain.T1824 {
					t.Errorf("getWeather().ChanceOfRain.T1824 = %v, want %v", got.ChanceOfRain.T1824, tt.want.ChanceOfRain.T1824)
				}
			}
		})
	}
}

type localRoundTripper struct {
	handler http.Handler
}

func (localRoundTripper localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ressponseRecorder := httptest.NewRecorder()
	localRoundTripper.handler.ServeHTTP(ressponseRecorder, req)
	return ressponseRecorder.Result(), nil
}

func TestPostSlack(t *testing.T) {
	type args struct {
		message string
	}

	tests := []struct {
		name   string
		apiRes string
		args   args
		want   string
	}{
		{
			name:   "postSlackIsOk",
			apiRes: "testdata/ok.json",
			args:   args{"message"},
			want:   "",
		},
		{
			name:   "postSlackIsError",
			apiRes: "testdata/error.json",
			args:   args{"message"},
			want:   "can not post. too_many_attachments",
		},
	}
	for _, tt := range tests {
		ts := slacktest.NewTestServer(func(c slacktest.Customize) {
			c.Handle("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
				res, _ := testdata.ReadFile(tt.apiRes)
				w.Write(res)
			})
		})
		ts.Start()
		client := slack.New("testToken", slack.OptionAPIURL(ts.GetAPIURL()))
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defaultFlags := log.Flags()
			log.SetFlags(0)
			defer func() {
				log.SetOutput(os.Stderr)
				log.SetFlags(defaultFlags)
				buf.Reset()
			}()

			SlackClient{client}.postSlack(tt.args.message)

			gotPrint := strings.TrimRight(buf.String(), "\n")
			if gotPrint != tt.want {
				t.Errorf("postSlack() = %v, want %v", gotPrint, tt.want)
			}
		})
	}
}
