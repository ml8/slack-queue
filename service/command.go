package service

import (
	"github.com/slack-go/slack"

	"net/http"
	"time"
)

func Defaults(api *slack.Client) (commands map[string]Command) {
	perms := &PermissionCheckerImpl{api: api}
	commands = make(map[string]Command)
	commands["/enqueue"] = &PutCommand{api}
	commands["/list"] = &ListCommand{api, perms}
	commands["/dequeue"] = &TakeCommand{api, perms}
	return
}

type Command interface {
	Handle(cmd *slack.SlashCommand, s *Service, w http.ResponseWriter) (err error)
}

type PermissionChecker interface {
	IsAdmin(user *slack.User) (ok bool, err error)
}

type PermissionCheckerImpl struct {
	adminChan string
	api       *slack.Client
	userCache []*slack.User
	cacheAge  time.Time
}

func (p *PermissionCheckerImpl) IsAdmin(user *slack.User) (ok bool, err error) {
	// TODO
	ok = true
	return
}

type ListCommand struct {
	api   *slack.Client
	perms PermissionChecker
}

type PutCommand struct {
	api *slack.Client
}

type TakeCommand struct {
	api   *slack.Client
	perms PermissionChecker
}
