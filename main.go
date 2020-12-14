package main

import (
	"github.com/matthewlang/slack-queue/service"
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

var srv *service.Service
var api *slack.Client
var perms service.PermissionChecker
var cmds map[string]service.Command
var actions map[string]service.Action

// Flags
var (
	oauth         string // OAuth token
	signingSecret string // Application signing secret
	clientSecret  string // Application client secret
	port          string // Port to listen on
	adminChannel  string // Channel containing admin users
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

	cmd, ok := cmds[s.Command]
	glog.Infof("Command parsed as %v for %v", s.Command, s)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cmd.Handle(&s, srv, w)
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
	js = strings.TrimPrefix(js, "payload=")
	var cb slack.InteractionCallback
	if err := json.Unmarshal([]byte(js), &cb); err != nil {
		glog.Errorf("Error unmarshalling callback: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler, ok := actions[cb.ActionID]
	if !ok {
		glog.Errorf("Unknown action type: %v", cb.ActionID)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	handler.Handle(&cb, srv, w)
}

func main() {
	flag.StringVar(&oauth, "oauth", "", "OAuth Token")
	flag.StringVar(&signingSecret, "ssecret", "", "Application signing secret")
	flag.StringVar(&clientSecret, "csecret", "", "Application client secret")
	flag.StringVar(&port, "p", ":1000", "Port to listen on")
	flag.StringVar(&adminChannel, "authChannel", "", "Channel containing admin users")
	flag.StringVar(&cmdUrl, "cmdUrl", "/slash", "URL to receive slash commands (e.g., '/slash' or '/receive', etc.)")
	flag.StringVar(&actionUrl, "actionUrl", "/action", "URL to receive actions")

	// TODO remove
	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	flag.Parse()

	glog.Infof("Starting on port %v ...", port)

	// TODO this needs to be an object.
	api = slack.New(oauth)
	srv = service.InMemoryTS(api)
	perms = service.MakeChannelPermissionChecker(api, adminChannel)
	cmds = service.DefaultCommands(api, perms)
	actions = service.DefaultActions(api, perms)

	http.HandleFunc(cmdUrl, forwardCmd)
	http.HandleFunc(actionUrl, forwardAction)

	glog.Infof("Listening...")
	http.ListenAndServe(port, nil)
}
