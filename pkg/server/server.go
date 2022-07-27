package server

import (
	"github.com/ml8/slack-queue/pkg/persister"
	"github.com/ml8/slack-queue/pkg/service"
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
	servers      map[string]*Server
	api          *slack.Client
	admin        service.AdminInterface
	command      string
	commandNames service.CommandNames
	persist      persister.Persister
}

func CreateServerGroup(api *slack.Client, admin service.AdminInterface, command string, commandNames service.CommandNames, persist persister.Persister) *ServerGroup {
	return &ServerGroup{
		servers:      make(map[string]*Server),
		api:          api,
		admin:        admin,
		command:      command,
		commandNames: commandNames,
		persist:      persist}
}

type Server struct {
	api       *slack.Client
	service   *service.QueueService
	admin     service.AdminInterface
	commands  map[string]service.Command
	actions   map[string]service.Action
	adminChan string
}

type ServerState struct {
	ChannelID string `json:"ChannelID"`
	AdminChan string `json:"AdminChan"`
}

type ServerGroupState struct {
	States []ServerState `json:"States"`
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

func (sg *ServerGroup) Persist() {
	if sg.persist == nil {
		return
	}
	glog.Infof("Persisting server list...")
	state := make([]ServerState, len(sg.servers))
	i := 0
	for key := range sg.servers {
		glog.Infof("%v", key)
		state[i] = ServerState{
			ChannelID: key,
			AdminChan: sg.servers[key].adminChan}
		i += 1
	}
	sgstate := ServerGroupState{state}
	glog.Infof("%d", sgstate.States)
	sg.persist.Write(sgstate)
}

func (sg *ServerGroup) Recover() {
	if sg.persist == nil {
		glog.Infof("Nothing to recover, using in-memory state.")
		return
	}
	glog.Infof("Recovering server list...")
	sgstate := ServerGroupState{}
	sg.persist.Read(&sgstate)
	glog.Infof("Recovered %d servers.", len(sgstate.States))
	for _, state := range sgstate.States {
		glog.Infof("Creating server for channel %v with admin channel %v", state.ChannelID, state.AdminChan)
		persist := persister.FilePersister{Fn: sg.persist.Id() + "-" + state.AdminChan}
		srv := service.PersistentTS(sg.api, persist)
		srv.Recover()

		admin := service.AdminInterfaceFromChannel(sg.api, state.AdminChan)
		sg.servers[state.ChannelID] = &Server{
			api:       sg.api,
			service:   srv,
			admin:     admin,
			commands:  service.DefaultCommands(sg.api, admin, sg.commandNames),
			actions:   service.DefaultActions(sg.api, admin),
			adminChan: state.AdminChan}
	}
}

// TODO this code is a mess

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
	var persist persister.Persister
	if sg.persist != nil {
		persist = persister.FilePersister{Fn: sg.persist.Id() + "-" + cmd.ChannelID}
	}
	sg.servers[cmd.ChannelID] = &Server{
		api:       sg.api,
		service:   service.PersistentTS(sg.api, persist),
		admin:     admin,
		commands:  service.DefaultCommands(sg.api, admin, sg.commandNames),
		actions:   service.DefaultActions(sg.api, admin),
		adminChan: channel}
	sg.api.PostMessage(cmd.ChannelID,
		slack.MsgOptionText("Queue created for channel.", false))
	sg.Persist()
}

func (sg *ServerGroup) rm(cmd *slack.SlashCommand, action string) {
	sg.Lock()
	defer sg.Unlock()

	// Check if it already exists.
	_, ok := sg.servers[cmd.ChannelID]

	if ok {
		delete(sg.servers, cmd.ChannelID)
		sg.Persist()
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
