package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"fmt"
	"net/http"
	"time"
)

func sendMatchDM(user *slack.User, admin *slack.User, msg string, api *slack.Client) (err error) {
	txt := fmt.Sprintf(
		"Hello %s! You've been matched with %s. Would you like to start a Zoom call?",
		user.RealName, admin.RealName)
	if msg != "" {
		txt = fmt.Sprintf("%s Topic: %s", txt, msg)
	}
	params := &slack.OpenConversationParameters{Users: []string{user.ID, admin.ID}}
	c, _, _, err := api.OpenConversation(params)
	if err != nil {
		return
	}
	if msg != "" {
		_, err = api.SetTopicOfConversation(c.ID, msg)
	}
	_, _, err = api.PostMessage(c.ID, slack.MsgOptionText(txt, false))
	return
}

func (a *TakeAction) Handle(action *slack.InteractionCallback, s *QueueService, w http.ResponseWriter) {
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
		glog.Errorf("Error dequeueing a request from %v (%v): %v", user.ID, user.Name, err)
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

	fu, err := a.ul.Lookup(user.ID)
	if err == nil {
		user = fu
	}

	err = sendMatchDM(resp.User, user, resp.Metadata, a.api)
	if err != nil {
		glog.Errorf("Error sending match message: %+v", err)
	}

	wt := time.Now().Sub(resp.Timestamp)
	str := fmt.Sprintf("%s dequeued %s (wait time %v)", userToLink(user), userToLink(resp.User), wt)
	cerr := a.perms.SendAdminMessage(str)
	if cerr != nil {
		glog.Errorf("Error sending admin message for dequeue of %v by %v: %v", resp.User.Name, action.User.Name, cerr)
	}

	return
}
