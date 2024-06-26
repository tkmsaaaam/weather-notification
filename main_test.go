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
	type wants struct {
		message *string
		print   string
	}
	weatherTokyo, _ := testdata.ReadFile("testdata/weather/TOKYO.json")
	weatherIsOk := "\n----------\n日時:2023/05/01 17:00:00\n概要:晴れています。\n夜は月が見えるでしょう。\n\n最低気温:0\n最高気温:30\n\n00-06:--%\n06-12:00%\n12-18:50%\n18-24:70%\n----------\n"
	tests := []struct {
		name   string
		apiRes res
		want   wants
	}{
		{
			name:   "watherIsOk",
			apiRes: res{status: 200, body: string(weatherTokyo)},
			want:   wants{message: &weatherIsOk, print: ""},
		},
		{
			name:   "apiIsError",
			apiRes: res{status: 500, body: "Internal Server Error"},
			want:   wants{message: nil, print: "Error Request API weather-api-go: request is failed. <nil>"},
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
			got := weatherClient.getWeather()
			if got != tt.want.message {
				if *got != *tt.want.message {
					t.Errorf("getWeather() = %v, want %v", *got, *tt.want.message)
				}
			}
			gotPrint := strings.TrimRight(buf.String(), "\n")
			if gotPrint != tt.want.print {
				t.Errorf("log = %v, want %v", gotPrint, tt.want.print)
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
