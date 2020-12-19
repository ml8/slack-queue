package main

import (
	"github.com/matthewlang/slack-queue/persister"
	"github.com/matthewlang/slack-queue/server"
	"github.com/matthewlang/slack-queue/service"
	"github.com/slack-go/slack"

	"github.com/golang/glog"

	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var api *slack.Client
var servers *server.ServerGroup

// Flags
var (
	oauth             string // OAuth token
	signingSecret     string // Application signing secret
	clientSecret      string // Application client secret
	port              string // Port to listen on
	cmdUrl            string // URL to receive slash commands
	actionUrl         string // URL to receive interactions
	authChannel       string // Channel of members permitted to create queues.
	managementCommand string // Command to manage queues.
	stateFilename     string // File to store persistent state.
	listCommand       string // Slash command for list
	putCommand        string // Slash command for put
	takeCommand       string // Slash command for take
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

	if s.Command == managementCommand {
		servers.Manage(&s, w)
		return
	}

	srv, ok := servers.Lookup(s.ChannelID)
	if !ok {
		glog.Infof("No server for channel %s (%s)", s.ChannelID, s.ChannelName)
		_, _, _ = api.PostMessage(s.ChannelID,
			slack.MsgOptionText(
				fmt.Sprintf("No queue exists for channel %s, use %s to create one.", s.ChannelName, managementCommand), false),
			slack.MsgOptionPostEphemeral(s.UserID))
		w.WriteHeader(http.StatusOK)
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
	srv, ok := servers.Lookup(cb.Channel.ID)
	if !ok {
		glog.Errorf("Received interaction for unserved channel %s (%s)", cb.Channel.ID, cb.Channel.Name)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	srv.ForwardAction(&cb, w)
}

func slashify(s string) string {
	if s[0] != '/' {
		return "/" + s
	}
	return s
}

func main() {
	flag.StringVar(&oauth, "oauth", "", "OAuth Token")
	flag.StringVar(&signingSecret, "ssecret", "", "Application signing secret")
	flag.StringVar(&clientSecret, "csecret", "", "Application client secret")
	flag.StringVar(&port, "p", ":1000", "Port to listen on")
	flag.StringVar(&cmdUrl, "cmdUrl", "/slash", "URL to receive slash commands (e.g., '/slash' or '/receive', etc.)")
	flag.StringVar(&actionUrl, "actionUrl", "/action", "URL to receive actions")
	flag.StringVar(&authChannel, "authChannel", "", "Channel authorized to create queues, empty means anyone can create a queue.")
	flag.StringVar(&managementCommand, "managementCommand", "queue", "Command used to manage queues.")
	flag.StringVar(&stateFilename, "stateFilename", "", "Root filename for persistent state.")
	flag.StringVar(&listCommand, "listCommand", "list", "Name of list slash command.")
	flag.StringVar(&putCommand, "putCommand", "enqueue", "Name of list slash command.")
	flag.StringVar(&takeCommand, "takeCommand", "dequeue", "Name of take slash command.")

	flag.Parse()

	glog.Infof("Starting on port %v ...", port)

	if managementCommand == "" {
		glog.Fatalf("Must supply a management command.")
	}

	managementCommand = slashify(managementCommand)
	listCommand = slashify(listCommand)
	putCommand = slashify(putCommand)
	takeCommand = slashify(takeCommand)

	glog.Infof("Using %s for management commands.", managementCommand)

	api = slack.New(oauth)

	var persist persister.Persister
	if stateFilename != "" {
		glog.Infof("Using %v for persistence.", stateFilename)
		persist = persister.FilePersister{Fn: stateFilename}
	} else {
		glog.Infof("Using in-memory state.")
	}

	servers = server.CreateServerGroup(
		api,
		service.AdminInterfaceFromChannel(api, authChannel),
		managementCommand,
		service.CommandNames{List: listCommand, Put: putCommand, Take: takeCommand},
		persist)

	servers.Recover()

	http.HandleFunc(cmdUrl, forwardCmd)
	http.HandleFunc(actionUrl, forwardAction)

	glog.Infof("Listening...")
	http.ListenAndServe(port, nil)
}
