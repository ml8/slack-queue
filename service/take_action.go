package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"fmt"
	"net/http"
	"time"
)

func (a *TakeAction) Handle(action *slack.InteractionCallback, s *Service, w http.ResponseWriter) {
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
	// Remove is a block action and should be in the actions for this callback.
	for _, act := range action.ActionCallback.BlockActions {
		if ParseAction(act.ActionID) == takeActionName {
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

	glog.Infof("Dequeuing position %d with token %d", pos, token)

	req := &DequeueRequest{}
	resp := &DequeueResponse{}

	req.Place = pos

	err = s.Dequeue(req, resp)
	if err != nil {
		glog.Errorf("Error dequeueing a request from %v (%v): %v", action.User.ID, action.User.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Replace list with updated state.
	updateListInUI(action, s, a.api)

	// If no user was dequeued, stop.
	if resp.User == nil {
		return
	}

	wt := time.Now().Sub(resp.Timestamp)
	str := fmt.Sprintf("%s dequeued %s (wait time %v)", userToLink(&action.User), userToLink(resp.User), wt)
	cerr := a.perms.SendAdminMessage(str)
	if cerr != nil {
		glog.Errorf("Error sending admin message for dequeue of %v by %v: %v", resp.User.Name, action.User.Name, cerr)
	}

	return
}