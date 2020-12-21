package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"net/http"
)

func (a *MoveAction) Handle(action *slack.InteractionCallback, s *QueueService, w http.ResponseWriter) {
	user := &action.User
	ok, err := a.perms.IsAdmin(user)
	if err != nil {
		glog.Errorf("Error checking admin status of %v (%v): %v", user.ID, user.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		glog.Errorf("Permission denied to user %v (%v)", user.ID, user.Name)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var pos int
	var token int64
	var found bool
	var actName string
	// Move is a block action and should be in the actions for this callback.
	for _, act := range action.ActionCallback.BlockActions {
		actName = ParseAction(act.ActionID)
		if actName == upActionName || actName == downActionName {
			pos, token, err = ParseActionValue(act.Value)
			found = true
			if err != nil {
				glog.Error("Error parsing action value %v: %v", act.Value, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			break
		}
	}

	if !found {
		glog.Errorf("Take action not found for remove callback!")
	}

	glog.Infof("Moving position %d %s with token %d", pos, actName, token)

	npos := pos - 1
	if actName == downActionName {
		npos = pos + 1
	}

	req := &MoveRequest{Pos: pos, NPos: npos, Token: token}
	resp := &MoveResponse{}

	err = s.Move(req, resp)
	if err != nil {
		glog.Errorf("Error moving %d from a request by %v (%v): %v", pos, action.User.ID, action.User.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Replace list with updated state.
	updateListInUI(action, s, a.api)
}
