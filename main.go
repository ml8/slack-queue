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
	glog.V(2).Infof("Action callback:\n%v", js)
	var cb slack.InteractionCallback
	if err := json.Unmarshal([]byte(js), &cb); err != nil {
		glog.Errorf("Error unmarshalling callback: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	glog.V(2).Infof("Action callback:\n%v", js)

	var handler service.Action
	ok := false
	// Only looking for block actions; right now at most one per payload.
	for _, act := range cb.ActionCallback.BlockActions {
		handler, ok = actions[service.ParseAction(act.ActionID)]
		if ok {
			break
		}
	}

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

	str := `{"type":"block_actions","user":{"id":"U01H5A7EAJC","username":"langma","name":"langma","team_id":"T01G0NGGZK9"},"api_app_id":"A01GMKWN0TW","token":"jrqQqPPhXLV6faVcS0ueWG55","container":{"type":"message","message_ts":"1608059435.001000","channel_id":"C01GCCBL5GV","is_ephemeral":true},"trigger_id":"1589929845329.1544764577655.4f82db2690090418bf03f9b221f0a18e","team":{"id":"T01G0NGGZK9","domain":"langtestworkspace"},"enterprise":null,"is_enterprise_install":false,"channel":{"id":"C01GCCBL5GV","name":"queuetest"},"response_url":"https:\/\/hooks.slack.com\/actions\/T01G0NGGZK9\/1577299513202\/EkXGc3AD4Ha1u9Yj0FPUve8x","actions":[{"action_id":"remove","block_id":"actions_U01H5A7EAJC","text":{"type":"plain_text","text":"Remove","emoji":true},"value":"U01H5A7EAJC_1","type":"button","action_ts":"1608059436.541193"}]}`

	var cb slack.InteractionCallback
	err := json.Unmarshal([]byte(str), &cb)
	glog.Infof("str %v", str)
	glog.Infof("err %v", err)
	glog.Infof("cb %+v", cb.ActionCallback.BlockActions[0])

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
