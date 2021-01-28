package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"encoding/json"
	"fmt"
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
		userstr = fmt.Sprintf("Ok! Up next is %s.", userToLink(resp.User))
		if resp.Metadata != "" {
			userstr = fmt.Sprintf("%s Topic: %s", userstr, resp.Metadata)
		}
		timestr = fmt.Sprintf("Time spent in queue: %v", (time.Now().Sub(resp.Timestamp)))
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

func (c *TakeCommand) Handle(cmd *slack.SlashCommand, s *QueueService, w http.ResponseWriter) (err error) {
	// Check permission to list queue.
	user := &slack.User{ID: cmd.UserID, Name: cmd.UserName, TeamID: cmd.TeamID}
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

	if resp.User == nil {
		// No one was dequeued. Stop.
		return
	}

	fu, err := c.ul.Lookup(user.ID)
	if err == nil {
		user = fu
	}

	wt := time.Now().Sub(resp.Timestamp)
	str := fmt.Sprintf("%s dequeued %s (wait time %v)", userToLink(user), userToLink(resp.User), wt)
	cerr := c.perms.SendAdminMessage(str)
	if cerr != nil {
		glog.Errorf("Error sending admin message for dequeue of %v by %v: %v", resp.User.Name, cmd.UserName, cerr)
	}

	err = sendMatchDM(resp.User, user, resp.Metadata, c.api)
	if err != nil {
		glog.Errorf("Error sending match message: %+v", err)
	}

	return
}
