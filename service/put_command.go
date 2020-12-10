package service

import (
	"github.com/golang/glog"
	"github.com/slack-go/slack"

	"encoding/json"
	"fmt"
	"net/http"
)

func enqueueAsBlock(cmd *slack.SlashCommand, resp *EnqueueResponse) (b []byte) {
	var statusstr string
	if resp.Ok {
		statusstr = fmt.Sprintf("*Status:*\nOk! You're %d in the queue.", resp.Pos+1)
	} else {
		statusstr = fmt.Sprintf("*Status:*\nYou are already queued at position %d.", resp.Pos+1)
	}
	timestr := fmt.Sprintf("*Enqueued At:*\n%v", resp.Timestamp.Local())

	fields := make([]*slack.TextBlockObject, 2)
	fields[0] = slack.NewTextBlockObject("mrkdwn", statusstr, false, false)
	fields[1] = slack.NewTextBlockObject("mrkdwn", timestr, false, false)
	section := slack.NewSectionBlock(nil, fields, nil)

	msg := slack.NewBlockMessage(section)
	b, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		glog.Fatalf("Error marshalling json: %v", err)
	}
	return
}

func buildEnqueueAdminMessage(resp *EnqueueResponse) (blocks []slack.Block) {
	str := fmt.Sprintf("%s added to queue in position %d", resp.User.Name, resp.Pos+1)
	blocks = []slack.Block{slack.NewContextBlock("context", slack.NewTextBlockObject("mrkdwn", str, false, false))}
	return blocks
}

func (c *PutCommand) Handle(cmd *slack.SlashCommand, s *Service, w http.ResponseWriter) (err error) {
	// TODO Send message to auth channel
	req := &EnqueueRequest{}
	resp := &EnqueueResponse{}

	req.User = &slack.User{}
	req.User.ID = cmd.UserID
	req.User.Name = cmd.UserName

	err = s.Enqueue(req, resp)
	if err != nil {
		glog.Errorf("Error enqueueing %v (%v): %v", cmd.UserID, cmd.UserName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b := enqueueAsBlock(cmd, resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	cerr := c.perms.SendAdminMessage(buildEnqueueAdminMessage(resp)...)
	if cerr != nil {
		glog.Errorf("Error sending admin message for enqueue of %v: %v", cmd.UserName, cerr)
	}

	return
}
