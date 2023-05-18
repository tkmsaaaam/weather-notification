package main

import (
	"bytes"
	_ "embed"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slacktest"
)

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
