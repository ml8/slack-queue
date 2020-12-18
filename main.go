package main

import (
	"github.com/slack-go/slack"

	"github.com/golang/glog"

	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var api *slack.Client
var services map[string]*Server

// Flags
var (
	oauth         string // OAuth token
	signingSecret string // Application signing secret
	clientSecret  string // Application client secret
	port          string // Port to listen on
	cmdUrl        string // URL to receive slash commands
	actionUrl     string // URL to receive interactions
)

func forwardCmd(w http.ResponseWriter, r *http.Request) {
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
	glog.V(1).Infof("Command parsed as %v for %v", s.Command, s)

	srv, ok := services[s.ChannelID]
	if !ok {
		glog.Infof("No server for channel %s (%s), creating...", s.ChannelID, s.ChannelName)
		// XXX
		services[s.ChannelID] = CreateServer(api, "adminchan")
		return
	}
	srv.ForwardCommand(&s, w)

}

func forwardAction(w http.ResponseWriter, r *http.Request) {
	verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
	if err != nil {
		glog.Infof("Could not create verifier: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
	buff, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("Error reading request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	js, err := url.QueryUnescape(string(buff))
	if err != nil {
		glog.Errorf("Error unescaping body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	glog.V(2).Infof("Action callback:\n%v", js)
	js = strings.TrimPrefix(js, "payload=")
	var cb slack.InteractionCallback
	if err := json.Unmarshal([]byte(js), &cb); err != nil {
		glog.Errorf("Error unmarshalling callback: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	glog.V(2).Infof("Action callback:\n%v", js)

	// TODO: is this the correct channel, when is cb.Channel and
	// cb.Container.Channel different?
	srv, ok := services[cb.Channel.ID]
	if !ok {
		glog.Errorf("Received interaction for unserved channel %s (%s)", cb.Channel.ID, cb.Channel.Name)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	srv.ForwardAction(&cb, w)
}

func main() {
	flag.StringVar(&oauth, "oauth", "", "OAuth Token")
	flag.StringVar(&signingSecret, "ssecret", "", "Application signing secret")
	flag.StringVar(&clientSecret, "csecret", "", "Application client secret")
	flag.StringVar(&port, "p", ":1000", "Port to listen on")
	flag.StringVar(&cmdUrl, "cmdUrl", "/slash", "URL to receive slash commands (e.g., '/slash' or '/receive', etc.)")
	flag.StringVar(&actionUrl, "actionUrl", "/action", "URL to receive actions")

	flag.Parse()

	glog.Infof("Starting on port %v ...", port)

	api = slack.New(oauth)
	services = make(map[string]*Server)

	http.HandleFunc(cmdUrl, forwardCmd)
	http.HandleFunc(actionUrl, forwardAction)

	glog.Infof("Listening...")
	http.ListenAndServe(port, nil)
}
