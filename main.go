package main

import (
	"github.com/matthewlang/slackqueue/service"
	"github.com/slack-go/slack"

	"github.com/golang/glog"

	"flag"
	"io"
	"io/ioutil"
	"net/http"
)

var srv *service.Service
var api *slack.Client
var cmds map[string]service.Command

// Flags
var (
	oauth         string
	signingSecret string
	clientSecret  string
	port          string
)

func forward(w http.ResponseWriter, r *http.Request) {
	verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
	if err != nil {
		glog.Infof("Could not create verifier: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
	s, err := slack.SlashCommandParse(r)
	if err != nil {
		glog.Infof("Unauthorized: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	cmd, ok := cmds[s.Command]
	glog.Infof("Command parsed as %v for %v", s.Command, s)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cmd.Handle(&s, srv, w)
}

func main() {
	flag.StringVar(&oauth, "oauth", "", "OAuth Token")
	flag.StringVar(&signingSecret, "ssecret", "", "Application signing secret")
	flag.StringVar(&clientSecret, "csecret", "", "Application client secret")
	flag.StringVar(&port, "p", ":1000", "Port to listen on")
	flag.Set("logtostderr", "true")
	flag.Parse()

	glog.Infof("Starting on port %v ...", port)

	api = slack.New(oauth)
	srv = service.InMemoryTS(api)
	cmds = service.Defaults(api)

	http.HandleFunc("/slash", forward)

	glog.Infof("Listening...")
	http.ListenAndServe(port, nil)
}
