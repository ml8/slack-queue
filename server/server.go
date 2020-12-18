package server

import (
	"github.com/matthewlang/slack-queue/service"
	"github.com/slack-go/slack"

	"github.com/golang/glog"

	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const (
	CreateString = "create"
	DeleteString = "delete"
)

type ServerGroup struct {
	sync.Mutex
	servers map[string]*Server
	api     *slack.Client
	admin   service.AdminInterface
	command string
}

func CreateServerGroup(api *slack.Client, admin service.AdminInterface, command string) *ServerGroup {
	return &ServerGroup{
		servers: make(map[string]*Server),
		api:     api,
		admin:   admin,
		command: command}
}

type Server struct {
	api      *slack.Client
	service  *service.QueueService
	admin    service.AdminInterface
	commands map[string]service.Command
	actions  map[string]service.Action
}

func CreateServer(api *slack.Client, adminChannel string) (s *Server) {
	s = &Server{}
	s.api = api
	s.service = service.InMemoryTS(api)
	s.admin = service.MakeChannelAdminInterface(api, adminChannel)
	s.commands = service.DefaultCommands(api, s.admin)
	s.actions = service.DefaultActions(api, s.admin)
	return
}

func (s *Server) ForwardCommand(cmd *slack.SlashCommand, w http.ResponseWriter) {
	c, ok := s.commands[cmd.Command]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	c.Handle(cmd, s.service, w)
}

func (s *Server) ForwardAction(act *slack.InteractionCallback, w http.ResponseWriter) {
	var handler service.Action
	ok := false
	// Only looking for block actions; right now at most one per payload.
	for _, a := range act.ActionCallback.BlockActions {
		handler, ok = s.actions[service.ParseAction(a.ActionID)]
		if ok {
			break
		}
	}

	if !ok {
		glog.Errorf("Unknown action type: %v", act.ActionID)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	handler.Handle(act, s.service, w)
}

func (sg *ServerGroup) Lookup(id string) (srv *Server, found bool) {
	sg.Lock()
	defer sg.Unlock()
	srv, found = sg.servers[id]
	return
}

func parseCommand(msg string) (cmd string, rest string, err error) {
	parts := strings.Split(msg, " ")
	if len(parts) > 2 || len(parts) < 1 {
		err = errors.New("Too few/many arguments")
	}
	cmd = parts[0]
	if len(parts) > 1 {
		rest = parts[1]
	}
	return
}

func (sg *ServerGroup) usage(cmd *slack.SlashCommand, w http.ResponseWriter) {
	sg.api.PostMessage(cmd.ChannelID,
		slack.MsgOptionText(fmt.Sprintf("Usage: %s create|delete [adminChannelName]", sg.command), false),
		slack.MsgOptionPostEphemeral(cmd.UserID))
}

// TODO this code is a mess

func (sg *ServerGroup) add(cmd *slack.SlashCommand, action string, channel string) {
	sg.Lock()
	defer sg.Unlock()
	_, ok := sg.servers[cmd.ChannelID]

	// Check if it already exists.
	if ok {
		sg.api.PostMessage(cmd.ChannelID,
			slack.MsgOptionText("Queue already exists in this channel.", false),
			slack.MsgOptionPostEphemeral(cmd.UserID))
		return
	}

	// Create it.
	admin := service.AdminInterfaceFromChannel(sg.api, channel)
	sg.servers[cmd.ChannelID] = &Server{
		api:      sg.api,
		service:  service.InMemoryTS(sg.api),
		admin:    admin,
		commands: service.DefaultCommands(sg.api, admin),
		actions:  service.DefaultActions(sg.api, admin)}
	sg.api.PostMessage(cmd.ChannelID,
		slack.MsgOptionText("Queue created for channel.", false))
}

func (sg *ServerGroup) rm(cmd *slack.SlashCommand, action string) {
	sg.Lock()
	defer sg.Unlock()

	// Check if it already exists.
	_, ok := sg.servers[cmd.ChannelID]

	if ok {
		delete(sg.servers, cmd.ChannelID)
		sg.api.PostMessage(cmd.ChannelID,
			slack.MsgOptionText("Deleted this channel's queue.", false))
	} else {
		sg.api.PostMessage(cmd.ChannelID,
			slack.MsgOptionText("No queue exists in this channel.", false),
			slack.MsgOptionPostEphemeral(cmd.UserID))
	}
}

func (sg *ServerGroup) Manage(cmd *slack.SlashCommand, w http.ResponseWriter) {
	// Check permission
	user := &slack.User{ID: cmd.UserID, Name: cmd.UserName, TeamID: cmd.TeamID}
	ok, err := sg.admin.IsAdmin(user)
	if err != nil {
		glog.Errorf("Error checking admin status of %v: %v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		glog.Errorf("Permission denied for user %v (%v)", user.ID, user.Name)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)

	action, channel, err := parseCommand(cmd.Text)
	if err != nil {
		sg.usage(cmd, w)
	}

	glog.Infof("Processing request %v %v", action, channel)

	// Handle creation
	switch action {
	case CreateString:
		sg.add(cmd, action, channel)
	case DeleteString:
		sg.rm(cmd, action)
	default:
		sg.usage(cmd, w)
	}
}
