package service

import (
	"github.com/slack-go/slack"

	"net/http"
)

func DefaultActions(api *slack.Client, perms PermissionChecker) (actions map[string]Action) {
	actions = make(map[string]Action)
	return
}

func ParseAction(actionID string) string {
	return actionID
}

type Action interface {
	Handle(action *slack.InteractionCallback, s *Service, w http.ResponseWriter)
}
