package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"encoding/json"
	"net/http"
	"time"
)

func dequeueAsBlock(cmd *slack.SlashCommand, resp *DequeueResponse) (b []byte) {
	var userstr string
	var timestr string
	if resp.User == nil {
		userstr = "*Queue is empty.*"
		timestr = ""
	} else {
		userstr = "Ok! Up next is <slack://user?id=" + resp.User.ID + "|" + resp.User.Name + ">"
		timestr = "Time spent in queue: " + (time.Now().Sub(resp.Timestamp)).String()
	}

	fields := make([]*slack.TextBlockObject, 2)
	fields[0] = slack.NewTextBlockObject("mrkdwn", userstr, false, false)
	fields[1] = slack.NewTextBlockObject("mrkdwn", timestr, false, false)
	section := slack.NewSectionBlock(nil, fields, nil)

	msg := slack.NewBlockMessage(section)
	b, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		glog.Fatalf("Error marshalling json: %v", err)
	}
	return
}

func (c *TakeCommand) Handle(cmd *slack.SlashCommand, s *Service, w http.ResponseWriter) (err error) {
	// TODO Send message to auth channel
	// Check permission to list queue.
	user := &slack.User{ID: cmd.UserID}
	ok, err := c.perms.IsAdmin(user)
	if err != nil {
		glog.Errorf("Error checking admin status of %v (%v): %v", cmd.UserID, cmd.UserName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		glog.Errorf("Permission denied to user %v (%v)", cmd.UserID, cmd.UserName)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := &DequeueRequest{}
	resp := &DequeueResponse{}

	req.Place = 0

	err = s.Dequeue(req, resp)
	if err != nil {
		glog.Errorf("Error dequeueing a request from %v (%v): %v", cmd.UserID, cmd.UserName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	b := dequeueAsBlock(cmd, resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	return
}
