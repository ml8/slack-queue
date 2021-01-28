package service

import (
	"github.com/slack-go/slack"

	"net/http"
)

const (
	// TODO make flags or make a config file.
	takeActionName   = "take"
	removeActionName = "remove"
	upActionName     = "up"
	downActionName   = "down"
)

// TODO(#20): There is a ton of duplicate code between the dequeue action and command and
// the remove and dequeue actions. This should be refactored.

func DefaultActions(api *slack.Client, perms AdminInterface) (actions map[string]Action) {
	actions = make(map[string]Action)
	actions[removeActionName] = &RemoveAction{api, perms, &UserLookupImpl{api}}
	actions[takeActionName] = &TakeAction{api, perms, &UserLookupImpl{api}}
	actions[upActionName] = &MoveAction{api, perms}
	actions[downActionName] = &MoveAction{api, perms}
	return
}

func ParseAction(actionID string) string {
	return actionID
}

type Action interface {
	Handle(action *slack.InteractionCallback, s *QueueService, w http.ResponseWriter)
}

type RemoveAction struct {
	api   *slack.Client
	perms AdminInterface
	ul    UserLookup
}

type TakeAction struct {
	api   *slack.Client
	perms AdminInterface
	ul    UserLookup
}

type MoveAction struct {
	api   *slack.Client
	perms AdminInterface
}
