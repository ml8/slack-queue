package service

import (
	"github.com/slack-go/slack"

	"net/http"
)

const (
	takeActionName   = "take"
	removeActionName = "remove"
)

// TODO(#20): There is a ton of duplicate code between the dequeue action and command and
// the remove and dequeue actions. This should be refactored.

func DefaultActions(api *slack.Client, perms AdminInterface) (actions map[string]Action) {
	actions = make(map[string]Action)
	actions[removeActionName] = &RemoveAction{api, perms}
	actions[takeActionName] = &TakeAction{api, perms}
	return
}

func ParseAction(actionID string) string {
	return actionID
}

type Action interface {
	Handle(action *slack.InteractionCallback, s *Service, w http.ResponseWriter)
}

type RemoveAction struct {
	api   *slack.Client
	perms AdminInterface
}

type TakeAction struct {
	api   *slack.Client
	perms AdminInterface
}
