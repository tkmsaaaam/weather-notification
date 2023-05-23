package main

import (
	"bytes"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slacktest"
	"github.com/tkmsaaaam/weather-api-go"
)

//go:embed testdata/weather/TOKYO.json
var weatherTokyo []byte

func TestGetWeather(t *testing.T) {
	type res struct {
		status int
		body   string
	}
	type wants struct {
		message string
		print   string
	}
	tests := []struct {
		name   string
		apiRes res
		want   wants
	}{
		{
			name:   "watherIsOk",
			apiRes: res{status: 200, body: string(weatherTokyo)},
			want:   wants{message: "\n----------\n日時:2023/05/01 17:00:00\n概要:晴れています。\n夜は月が見えるでしょう。\n\n最低気温:0\n最高気温:30\n\n00-06:--%\n06-12:00%\n12-18:50%\n18-24:70%\n----------\n", print: ""},
		},
		{
			name:   "apiIsError",
			apiRes: res{status: 500, body: "Internal Server Error"},
			want:   wants{message: "\n----------\n日時:\n概要:\n最低気温:\n最高気温:\n\n00-06:\n06-12:\n12-18:\n18-24:\n----------\n", print: "json.Unmarshal err: invalid character 'I' looking for beginning of value\nError Request API"},
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

			orgStdout := os.Stdout
			defer func() {
				os.Stdout = orgStdout
			}()
			r, w, _ := os.Pipe()
			os.Stdout = w
			got := weatherClient.getWeather()
			w.Close()
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("failed to read buf: %v", err)
			}
			if got != tt.want.message {
				t.Errorf("add() = %v, want %v", got, tt.want.message)
			}
			gotPrint := strings.TrimRight(buf.String(), "\n")
			if gotPrint != tt.want.print {
				t.Errorf("add() = %v, want %v", gotPrint, tt.want.print)
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

//go:embed testdata/ok.json
var postSlackIsOk []byte

//go:embed testdata/error.json
var postSlackIsError []byte

func TestPostSlack(t *testing.T) {
	type args struct {
		message string
	}

	tests := []struct {
		name   string
		apiRes []byte
		args   args
		want   string
	}{
		{
			name:   "postSlackIsOk",
			apiRes: postSlackIsOk,
			args:   args{"message"},
			want:   "",
		},
		{
			name:   "postSlackIsError",
			apiRes: postSlackIsError,
			args:   args{"message"},
			want:   "too_many_attachments",
		},
	}
	for _, tt := range tests {
		ts := slacktest.NewTestServer(func(c slacktest.Customize) {
			c.Handle("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
				w.Write(tt.apiRes)
			})
		})
		ts.Start()
		client := slack.New("testToken", slack.OptionAPIURL(ts.GetAPIURL()))
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			orgStdout := os.Stdout
			defer func() {
				os.Stdout = orgStdout
			}()
			r, w, _ := os.Pipe()
			os.Stdout = w
			SlackClient{client}.postSlack(tt.args.message)
			w.Close()
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("failed to read buf: %v", err)
			}
			gotPrint := strings.TrimRight(buf.String(), "\n")
			if gotPrint != tt.want {
				t.Errorf("add() = %v, want %v", gotPrint, tt.want)
			}
		})
	}
}
