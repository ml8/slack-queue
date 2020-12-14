package service

import (
	"github.com/slack-go/slack"

	"net/http"
)

func DefaultCommands(api *slack.Client, perms PermissionChecker) (commands map[string]Command) {
	commands = make(map[string]Command)
	commands["/enqueue"] = &PutCommand{api, perms}
	commands["/list"] = &ListCommand{api, perms}
	commands["/dequeue"] = &TakeCommand{api, perms}
	return
}

type Command interface {
	Handle(cmd *slack.SlashCommand, s *Service, w http.ResponseWriter) (err error)
}

type ListCommand struct {
	api   *slack.Client
	perms PermissionChecker
}

type PutCommand struct {
	api   *slack.Client
	perms PermissionChecker
}

type TakeCommand struct {
	api   *slack.Client
	perms PermissionChecker
}
