package server

import (
	"github.com/matthewlang/slack-queue/service"
	"github.com/slack-go/slack"

	"github.com/golang/glog"

	"net/http"
)

type ServerGroup struct {
	servers map[string]*Server
	api     *slack.Client
	admin   service.AdminInterface
}

func CreateServerGroup(api *slack.Client, admin service.AdminInterface) *ServerGroup {
	return &ServerGroup{
		servers: make(map[string]*Server),
		api:     api,
		admin:   admin}
}

type Server struct {
	service  *service.Service
	api      *slack.Client
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
	srv, found = sg.servers[id]
	return
}

func (sg *ServerGroup) Manage(cmd *slack.SlashCommand, w http.ResponseWriter) {
}
