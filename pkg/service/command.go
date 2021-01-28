package service

import (
	"github.com/slack-go/slack"

	"net/http"
)

type CommandNames struct {
	Take string
	Put  string
	List string
}

func DefaultCommands(api *slack.Client, perms AdminInterface, names CommandNames) (commands map[string]Command) {
	commands = make(map[string]Command)
	commands[names.Put] = &PutCommand{api, perms, &UserLookupImpl{api}}
	commands[names.List] = &ListCommand{api, perms}
	commands[names.Take] = &TakeCommand{api, perms, &UserLookupImpl{api}}
	return
}

type Command interface {
	Handle(cmd *slack.SlashCommand, s *QueueService, w http.ResponseWriter) (err error)
}

type ListCommand struct {
	api   *slack.Client
	perms AdminInterface
}

type PutCommand struct {
	api   *slack.Client
	perms AdminInterface
	ul    UserLookup
}

type TakeCommand struct {
	api   *slack.Client
	perms AdminInterface
	ul    UserLookup
}
